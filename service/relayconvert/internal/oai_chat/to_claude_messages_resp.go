package oaichat

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/reasonmap"
	"github.com/samber/lo"
)

func generateStopBlock(index int) *dto.ClaudeResponse {
	return &dto.ClaudeResponse{
		Type:  "content_block_stop",
		Index: common.GetPointer[int](index),
	}
}

func buildClaudeUsageFromOpenAIUsage(oaiUsage *dto.Usage) *dto.ClaudeUsage {
	if oaiUsage == nil {
		return nil
	}
	if billingUsage := dto.CloneBillingUsage(oaiUsage.BillingUsage); billingUsage != nil && billingUsage.ClaudeUsage != nil {
		if billingUsage.Source == dto.BillingUsageSourceClaudeMessages || billingUsage.Semantic == dto.BillingUsageSemanticAnthropic {
			return billingUsage.ClaudeUsage
		}
	}
	billingUsage := dto.NewOpenAIChatBillingUsage(oaiUsage)
	if existingBillingUsage := dto.CloneBillingUsage(oaiUsage.BillingUsage); existingBillingUsage != nil && existingBillingUsage.OpenAIUsage != nil {
		if existingBillingUsage.Source == dto.BillingUsageSourceOAIChat ||
			existingBillingUsage.Source == dto.BillingUsageSourceOAIResponses ||
			existingBillingUsage.Semantic == dto.BillingUsageSemanticOpenAI {
			billingUsage = existingBillingUsage
		}
	}
	cacheCreation5m, cacheCreation1h := NormalizeCacheCreationSplit(
		oaiUsage.PromptTokensDetails.CachedCreationTokens,
		oaiUsage.ClaudeCacheCreation5mTokens,
		oaiUsage.ClaudeCacheCreation1hTokens,
	)
	cacheCreationTokens := oaiUsage.PromptTokensDetails.CacheCreationTokensTotal()
	inputTokens := oaiUsage.PromptTokens
	if oaiUsage.PromptTokensDetails.CacheWriteTokens > 0 {
		// OpenAI native cache-write usage counts cached and cache-write tokens
		// inside prompt_tokens, while Claude semantics reports input_tokens
		// excluding both. Both counts are unadjusted prefixes and may overlap,
		// so clamp a negative remainder at zero.
		inputTokens = oaiUsage.PromptTokens - oaiUsage.PromptTokensDetails.CachedTokens - cacheCreationTokens
		if inputTokens < 0 {
			inputTokens = 0
		}
	}
	usage := &dto.ClaudeUsage{
		InputTokens:              inputTokens,
		OutputTokens:             oaiUsage.CompletionTokens,
		CacheCreationInputTokens: cacheCreationTokens,
		CacheReadInputTokens:     oaiUsage.PromptTokensDetails.CachedTokens,
		BillingUsage:             billingUsage,
	}
	if cacheCreation5m > 0 || cacheCreation1h > 0 {
		usage.CacheCreation = &dto.ClaudeCacheCreationUsage{
			Ephemeral5mInputTokens: cacheCreation5m,
			Ephemeral1hInputTokens: cacheCreation1h,
		}
	}
	return usage
}

func NormalizeCacheCreationSplit(totalTokens int, tokens5m int, tokens1h int) (int, int) {
	remainder := lo.Max([]int{totalTokens - tokens5m - tokens1h, 0})
	return tokens5m + remainder, tokens1h
}

func appendClaudeToolCallDeltas(
	claudeResponses []*dto.ClaudeResponse,
	toolCalls []dto.ToolCallResponse,
	info *relaycommon.ClaudeConvertInfo,
) []*dto.ClaudeResponse {
	if info.ToolCallBlockIndexes == nil {
		info.ToolCallBlockIndexes = make(map[int]int)
	}
	if info.PendingToolCalls == nil {
		info.PendingToolCalls = make(map[int]dto.ToolCallResponse)
	}

	for i, toolCall := range toolCalls {
		sourceIndex := i
		if toolCall.Index != nil {
			sourceIndex = *toolCall.Index
		}

		blockIndex, started := info.ToolCallBlockIndexes[sourceIndex]
		if !started {
			if pending, ok := info.PendingToolCalls[sourceIndex]; ok {
				if toolCall.ID == "" {
					toolCall.ID = pending.ID
				}
				if toolCall.Function.Name == "" {
					toolCall.Function.Name = pending.Function.Name
				}
				toolCall.Function.Arguments = pending.Function.Arguments + toolCall.Function.Arguments
			}

			if toolCall.Function.Name == "" {
				info.PendingToolCalls[sourceIndex] = toolCall
				continue
			}

			blockIndex = info.Index
			if len(info.ToolCallBlockIndexes) > 0 {
				blockIndex++
			}
			info.ToolCallBlockIndexes[sourceIndex] = blockIndex
			info.Index = blockIndex
			delete(info.PendingToolCalls, sourceIndex)

			idx := blockIndex
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Index: &idx,
				Type:  "content_block_start",
				ContentBlock: &dto.ClaudeMediaMessage{
					Id:    toolCall.ID,
					Type:  "tool_use",
					Name:  toolCall.Function.Name,
					Input: map[string]interface{}{},
				},
			})
		}

		if toolCall.Function.Arguments != "" {
			idx := blockIndex
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Index: &idx,
				Type:  "content_block_delta",
				Delta: &dto.ClaudeMediaMessage{
					Type:        "input_json_delta",
					PartialJson: &toolCall.Function.Arguments,
				},
			})
		}
	}

	return claudeResponses
}

func StreamResponseOpenAI2Claude(openAIResponse *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) []*dto.ClaudeResponse {
	if info == nil {
		info = &relaycommon.RelayInfo{}
	}
	if info.ClaudeConvertInfo == nil {
		info.ClaudeConvertInfo = &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		}
	}
	if info.ClaudeConvertInfo.Done {
		return nil
	}

	var claudeResponses []*dto.ClaudeResponse
	// stopOpenBlocks emits the required content_block_stop event(s) for the currently open block(s)
	// according to Anthropic's SSE streaming state machine:
	// content_block_start -> content_block_delta* -> content_block_stop (per index).
	//
	// For text/thinking, there is at most one open block at info.ClaudeConvertInfo.Index.
	// For tools, OpenAI tool_calls can stream multiple parallel tool_use blocks (indexed from 0),
	// so we may have multiple open blocks and must stop each one explicitly.
	stopOpenBlocks := func() {
		switch info.ClaudeConvertInfo.LastMessagesType {
		case relaycommon.LastMessageTypeText, relaycommon.LastMessageTypeThinking:
			claudeResponses = append(claudeResponses, generateStopBlock(info.ClaudeConvertInfo.Index))
		case relaycommon.LastMessageTypeTools:
			blockIndexes := make([]int, 0, len(info.ClaudeConvertInfo.ToolCallBlockIndexes))
			for _, blockIndex := range info.ClaudeConvertInfo.ToolCallBlockIndexes {
				blockIndexes = append(blockIndexes, blockIndex)
			}
			sort.Ints(blockIndexes)
			for _, blockIndex := range blockIndexes {
				claudeResponses = append(claudeResponses, generateStopBlock(blockIndex))
			}
		}
	}
	// stopOpenBlocksAndAdvance closes the currently open block(s) and advances the content block index
	// to the next available slot for subsequent content_block_start events.
	//
	// This prevents invalid streams where a content_block_delta (e.g. thinking_delta) is emitted for an
	// index whose active content_block type is different (the typical cause of "Mismatched content block type").
	stopOpenBlocksAndAdvance := func() {
		if info.ClaudeConvertInfo.LastMessagesType == relaycommon.LastMessageTypeNone {
			return
		}
		stopOpenBlocks()
		switch info.ClaudeConvertInfo.LastMessagesType {
		case relaycommon.LastMessageTypeTools:
			if len(info.ClaudeConvertInfo.ToolCallBlockIndexes) > 0 {
				info.ClaudeConvertInfo.Index++
			}
			info.ClaudeConvertInfo.ToolCallBlockIndexes = nil
			info.ClaudeConvertInfo.PendingToolCalls = nil
		default:
			info.ClaudeConvertInfo.Index++
		}
		info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeNone
	}
	if info.SendResponseCount == 1 {
		msg := &dto.ClaudeMediaMessage{
			Id:    openAIResponse.Id,
			Model: openAIResponse.Model,
			Type:  "message",
			Role:  "assistant",
			Usage: &dto.ClaudeUsage{
				InputTokens:  info.GetEstimatePromptTokens(),
				OutputTokens: 0,
			},
		}
		msg.SetContent(make([]any, 0))
		claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
			Type:    "message_start",
			Message: msg,
		})
		//claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
		//	Type: "ping",
		//})
		if openAIResponse.IsToolCall() {
			info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeTools
			var toolCalls []dto.ToolCallResponse
			if len(openAIResponse.Choices) > 0 && len(openAIResponse.Choices[0].Delta.ToolCalls) > 0 {
				toolCalls = openAIResponse.Choices[0].Delta.ToolCalls
			} else {
				first := openAIResponse.GetFirstToolCall()
				if first != nil {
					toolCalls = []dto.ToolCallResponse{*first}
				}
			}
			claudeResponses = appendClaudeToolCallDeltas(claudeResponses, toolCalls, info.ClaudeConvertInfo)
		}
		// 判断首个响应是否存在内容（非标准的 OpenAI 响应）
		if len(openAIResponse.Choices) > 0 {
			reasoning := openAIResponse.Choices[0].Delta.GetReasoningContent()
			content := openAIResponse.Choices[0].Delta.GetContentString()

			if reasoning != "" {
				if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeThinking {
					stopOpenBlocksAndAdvance()
				}
				idx := info.ClaudeConvertInfo.Index
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Index: &idx,
					Type:  "content_block_start",
					ContentBlock: &dto.ClaudeMediaMessage{
						Type:     "thinking",
						Thinking: common.GetPointer[string](""),
					},
				})
				idx2 := idx
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Index: &idx2,
					Type:  "content_block_delta",
					Delta: &dto.ClaudeMediaMessage{
						Type:     "thinking_delta",
						Thinking: &reasoning,
					},
				})
				info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeThinking
			} else if content != "" {
				if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeText {
					stopOpenBlocksAndAdvance()
				}
				idx := info.ClaudeConvertInfo.Index
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Index: &idx,
					Type:  "content_block_start",
					ContentBlock: &dto.ClaudeMediaMessage{
						Type: "text",
						Text: common.GetPointer[string](""),
					},
				})
				idx2 := idx
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Index: &idx2,
					Type:  "content_block_delta",
					Delta: &dto.ClaudeMediaMessage{
						Type: "text_delta",
						Text: common.GetPointer[string](content),
					},
				})
				info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeText
			}
		}

		// 如果首块就带 finish_reason，需要立即发送停止块
		if len(openAIResponse.Choices) > 0 && openAIResponse.Choices[0].FinishReason != nil && *openAIResponse.Choices[0].FinishReason != "" {
			info.FinishReason = *openAIResponse.Choices[0].FinishReason
			stopOpenBlocks()
			oaiUsage := openAIResponse.Usage
			if oaiUsage == nil {
				oaiUsage = info.ClaudeConvertInfo.Usage
			}
			if oaiUsage != nil {
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Type:  "message_delta",
					Usage: buildClaudeUsageFromOpenAIUsage(oaiUsage),
					Delta: &dto.ClaudeMediaMessage{
						StopReason: common.GetPointer[string](stopReasonOpenAI2Claude(info.FinishReason)),
					},
				})
			}
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Type: "message_stop",
			})
			info.ClaudeConvertInfo.Done = true
		}
		return claudeResponses
	}

	if len(openAIResponse.Choices) == 0 {
		// Some OpenAI-compatible upstreams end with a usage-only SSE chunk.
		oaiUsage := openAIResponse.Usage
		if oaiUsage == nil {
			oaiUsage = info.ClaudeConvertInfo.Usage
		}
		if oaiUsage != nil {
			stopOpenBlocks()
			stopReason := stopReasonOpenAI2Claude(info.FinishReason)
			if stopReason == "" {
				stopReason = "end_turn"
			}
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Type:  "message_delta",
				Usage: buildClaudeUsageFromOpenAIUsage(oaiUsage),
				Delta: &dto.ClaudeMediaMessage{
					StopReason: common.GetPointer[string](stopReason),
				},
			})
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Type: "message_stop",
			})
			info.ClaudeConvertInfo.Done = true
		}
		return claudeResponses
	} else {
		chosenChoice := openAIResponse.Choices[0]
		doneChunk := chosenChoice.FinishReason != nil && *chosenChoice.FinishReason != ""
		if doneChunk {
			info.FinishReason = *chosenChoice.FinishReason
			oaiUsage := openAIResponse.Usage
			if oaiUsage == nil {
				oaiUsage = info.ClaudeConvertInfo.Usage
				// Some upstreams emit finish_reason first, then send a final usage-only chunk.
				// Defer closing until usage is available so the final message_delta carries it.
				return claudeResponses
			}
		}

		var claudeResponse dto.ClaudeResponse
		var isEmpty bool
		claudeResponse.Type = "content_block_delta"
		if len(chosenChoice.Delta.ToolCalls) > 0 {
			toolCalls := chosenChoice.Delta.ToolCalls
			if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeTools {
				stopOpenBlocksAndAdvance()
			}
			info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeTools
			claudeResponses = appendClaudeToolCallDeltas(claudeResponses, toolCalls, info.ClaudeConvertInfo)
		} else {
			reasoning := chosenChoice.Delta.GetReasoningContent()
			textContent := chosenChoice.Delta.GetContentString()
			if reasoning != "" || textContent != "" {
				if reasoning != "" {
					if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeThinking {
						stopOpenBlocksAndAdvance()
						idx := info.ClaudeConvertInfo.Index
						claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
							Index: &idx,
							Type:  "content_block_start",
							ContentBlock: &dto.ClaudeMediaMessage{
								Type:     "thinking",
								Thinking: common.GetPointer[string](""),
							},
						})
					}
					info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeThinking
					claudeResponse.Delta = &dto.ClaudeMediaMessage{
						Type:     "thinking_delta",
						Thinking: &reasoning,
					}
				} else {
					if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeText {
						stopOpenBlocksAndAdvance()
						idx := info.ClaudeConvertInfo.Index
						claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
							Index: &idx,
							Type:  "content_block_start",
							ContentBlock: &dto.ClaudeMediaMessage{
								Type: "text",
								Text: common.GetPointer[string](""),
							},
						})
					}
					info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeText
					claudeResponse.Delta = &dto.ClaudeMediaMessage{
						Type: "text_delta",
						Text: common.GetPointer[string](textContent),
					}
				}
			} else {
				isEmpty = true
			}
		}

		claudeResponse.Index = common.GetPointer[int](info.ClaudeConvertInfo.Index)
		if !isEmpty && claudeResponse.Delta != nil {
			claudeResponses = append(claudeResponses, &claudeResponse)
		}

		if doneChunk || info.ClaudeConvertInfo.Done {
			stopOpenBlocks()
			oaiUsage := openAIResponse.Usage
			if oaiUsage == nil {
				oaiUsage = info.ClaudeConvertInfo.Usage
			}
			if oaiUsage != nil {
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Type:  "message_delta",
					Usage: buildClaudeUsageFromOpenAIUsage(oaiUsage),
					Delta: &dto.ClaudeMediaMessage{
						StopReason: common.GetPointer[string](stopReasonOpenAI2Claude(info.FinishReason)),
					},
				})
			}
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Type: "message_stop",
			})
			info.ClaudeConvertInfo.Done = true
			return claudeResponses
		}
	}

	return claudeResponses
}

func ResponseOpenAI2Claude(openAIResponse *dto.OpenAITextResponse, info *relaycommon.RelayInfo) *dto.ClaudeResponse {
	var stopReason string
	contents := make([]dto.ClaudeMediaMessage, 0)
	claudeResponse := &dto.ClaudeResponse{
		Id:    openAIResponse.Id,
		Type:  "message",
		Role:  "assistant",
		Model: openAIResponse.Model,
	}
	for _, choice := range openAIResponse.Choices {
		stopReason = stopReasonOpenAI2Claude(choice.FinishReason)
		textContent := choice.Message.StringContent()
		toolCalls := choice.Message.ParseToolCalls()
		if textContent != "" || len(toolCalls) == 0 {
			claudeContent := dto.ClaudeMediaMessage{}
			claudeContent.Type = "text"
			claudeContent.SetText(textContent)
			contents = append(contents, claudeContent)
		}
		for _, toolUse := range toolCalls {
			claudeContent := dto.ClaudeMediaMessage{}
			claudeContent.Type = "tool_use"
			claudeContent.Id = toolUse.ID
			claudeContent.Name = toolUse.Function.Name
			mapParams := map[string]interface{}{}
			if strings.TrimSpace(toolUse.Function.Arguments) != "" {
				var parsed map[string]interface{}
				if err := common.Unmarshal([]byte(toolUse.Function.Arguments), &parsed); err == nil && parsed != nil {
					mapParams = parsed
				}
			}
			claudeContent.Input = mapParams
			contents = append(contents, claudeContent)
		}
	}
	claudeResponse.Content = contents
	claudeResponse.StopReason = stopReason
	claudeResponse.Usage = buildClaudeUsageFromOpenAIUsage(&openAIResponse.Usage)

	return claudeResponse
}

func stopReasonOpenAI2Claude(reason string) string {
	return reasonmap.OpenAIFinishReasonToClaudeStopReason(reason)
}
