// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package service

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"sync"
)

// Ensure, that OpenAIClientMock does implement OpenAIClient.
// If this is not the case, regenerate this file with moq.
var _ OpenAIClient = &OpenAIClientMock{}

// OpenAIClientMock is a mock implementation of OpenAIClient.
//
// 	func TestSomethingThatUsesOpenAIClient(t *testing.T) {
//
// 		// make and configure a mocked OpenAIClient
// 		mockedOpenAIClient := &OpenAIClientMock{
// 			CreateChatCompletionFunc: func(contextMoqParam context.Context, chatCompletionRequest openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
// 				panic("mock out the CreateChatCompletion method")
// 			},
// 		}
//
// 		// use mockedOpenAIClient in code that requires OpenAIClient
// 		// and then make assertions.
//
// 	}
type OpenAIClientMock struct {
	// CreateChatCompletionFunc mocks the CreateChatCompletion method.
	CreateChatCompletionFunc func(contextMoqParam context.Context, chatCompletionRequest openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)

	// calls tracks calls to the methods.
	calls struct {
		// CreateChatCompletion holds details about calls to the CreateChatCompletion method.
		CreateChatCompletion []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
			// ChatCompletionRequest is the chatCompletionRequest argument value.
			ChatCompletionRequest openai.ChatCompletionRequest
		}
	}
	lockCreateChatCompletion sync.RWMutex
}

// CreateChatCompletion calls CreateChatCompletionFunc.
func (mock *OpenAIClientMock) CreateChatCompletion(contextMoqParam context.Context, chatCompletionRequest openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if mock.CreateChatCompletionFunc == nil {
		panic("OpenAIClientMock.CreateChatCompletionFunc: method is nil but OpenAIClient.CreateChatCompletion was just called")
	}
	callInfo := struct {
		ContextMoqParam       context.Context
		ChatCompletionRequest openai.ChatCompletionRequest
	}{
		ContextMoqParam:       contextMoqParam,
		ChatCompletionRequest: chatCompletionRequest,
	}
	mock.lockCreateChatCompletion.Lock()
	mock.calls.CreateChatCompletion = append(mock.calls.CreateChatCompletion, callInfo)
	mock.lockCreateChatCompletion.Unlock()
	return mock.CreateChatCompletionFunc(contextMoqParam, chatCompletionRequest)
}

// CreateChatCompletionCalls gets all the calls that were made to CreateChatCompletion.
// Check the length with:
//     len(mockedOpenAIClient.CreateChatCompletionCalls())
func (mock *OpenAIClientMock) CreateChatCompletionCalls() []struct {
	ContextMoqParam       context.Context
	ChatCompletionRequest openai.ChatCompletionRequest
} {
	var calls []struct {
		ContextMoqParam       context.Context
		ChatCompletionRequest openai.ChatCompletionRequest
	}
	mock.lockCreateChatCompletion.RLock()
	calls = mock.calls.CreateChatCompletion
	mock.lockCreateChatCompletion.RUnlock()
	return calls
}