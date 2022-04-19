//go:build !arm64
// +build !arm64

package asr

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
)

var (
	azureConfig *speech.SpeechConfig
)

func initConfig() error {
	var err error

	if subscription := os.Getenv("AZURE_KEY"); subscription != "" {
		if region := os.Getenv("AZURE_REGION"); region != "" {
			azureConfig, err = speech.NewSpeechConfigFromSubscription(subscription, region)
			if err != nil {
				return err
			}
		} else {
			msg := "Could not initialize ASR with env ******AZURE_REGION******"
			fmt.Printf(msg)
		}
	} else {
		msg := "Could not initialize ASR with env ******AZURE_KEY******"
		fmt.Printf(msg)
	}

	return nil
}

func init() {
	err := initConfig()
	if err != nil {
		fmt.Printf("(Azure Client) failed to initialize: %v", err)
	}
}

type AzureASR struct{}

func (*AzureASR) Recognize(audio_bytes []byte) (transcript string, err error) {
	azureTimeout := 25
	languageCode := "en-IN"

	azureConfig.SetSpeechRecognitionLanguage(languageCode)
	azureConfig.SetOutputFormat(common.Detailed)

	tmpFile, err := ioutil.TempFile(os.TempDir(), "azure-*.wav")
	if err != nil {
		msg := "(Azure Client) Failed to create temp file"
		fmt.Printf(msg)
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.Write(audio_bytes); err != nil {
		msg := "(Azure Client) Failed to write to temp file"
		fmt.Printf(msg)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		msg := "(Azure Client) Failed to close temp file"
		fmt.Printf(msg)
		return "", err
	}

	audioConfig, err := audio.NewAudioConfigFromWavFileInput(tmpFile.Name())
	if err != nil {
		msg := "(Azure Client) Failed to init audio config from input file"
		fmt.Printf(msg)
		return "", err
	}
	defer audioConfig.Close()

	speechRecognizer, err := speech.NewSpeechRecognizerFromConfig(azureConfig, audioConfig)
	if err != nil {
		msg := "(Azure Client) Failed to create new SpeechRecognizer from config"
		fmt.Printf(msg)
		return "", err
	}
	defer speechRecognizer.Close()

	speechRecognizer.SessionStarted(azureSessionHandler)
	speechRecognizer.SessionStopped(azureSessionHandler)

	task := speechRecognizer.RecognizeOnceAsync()
	var outcome speech.SpeechRecognitionOutcome
	select {
	case outcome = <-task:
	case <-time.After(time.Duration(azureTimeout) * time.Second):
		msg := "(Azure Client) Timed out"
		fmt.Printf(msg)
		return "", err
	}
	defer outcome.Close()

	if outcome.Error != nil {
		msg := "(Azure Client) Recognize failed"
		fmt.Printf(msg)
		return "", err
	}

	transcript = outcome.Result.Text
	return
}

func (*AzureASR) StreamingRecognize(stream chan []byte) (transcript string, err error) {
	languageCode := "en-IN"
	sampleRateHertz := 8000

	azureConfig.SetSpeechRecognitionLanguage(languageCode)
	azureConfig.SetOutputFormat(common.Detailed)

	audioFormat, _ := audio.GetWaveFormatPCM(uint32(sampleRateHertz), 16, 1)

	azure_stream, err := audio.CreatePushAudioInputStreamFromFormat(audioFormat)
	if err != nil {
		msg := "(Azure Client) Failed to create PushAudioInputStream"
		fmt.Printf(msg)
		return "", err
	}

	audioConfig, err := audio.NewAudioConfigFromStreamInput(azure_stream)
	if err != nil {
		msg := "(Azure Client) Failed to init audio config from input stream"
		fmt.Printf(msg)
		return "", err
	}
	defer audioConfig.Close()

	speechRecognizer, err := speech.NewSpeechRecognizerFromConfig(azureConfig, audioConfig)
	if err != nil {
		msg := "(Azure Client) Failed to create new SpeechRecognizer from config"
		fmt.Printf(msg)
		return "", err
	}
	defer speechRecognizer.Close()

	// define callbacks
	speechRecognizer.SessionStarted(azureSessionHandler)
	speechRecognizer.SessionStopped(azureSessionHandler)
	speechRecognizer.Recognizing(azureRecognitionHandler)

	speechRecognizer.Recognized(func(event speech.SpeechRecognitionEventArgs) {
		defer event.Close()
		fmt.Println("Result: ", event.Result)
		if transcript == "" {
			transcript = event.Result.Text
		} else {
			transcript = transcript + " " + event.Result.Text
		}
	})

	speechRecognizer.Canceled(func(event speech.SpeechRecognitionCanceledEventArgs) {
		defer event.Close()
		fmt.Printf("(Azure Client) Received a cancellation: %v", event.ErrorDetails)
	})

	// start recognition
	speechRecognizer.StartContinuousRecognitionAsync()

	for {
		payload, ok := <-stream

		if !ok { // When EOF is reached, close streams and receive results
			fmt.Printf("(Azure Client) StreamingRecognize: received end of stream, closing stream")
			azure_stream.Close()

			// stop recognition
			speechRecognizer.StopContinuousRecognitionAsync()
			if err != nil {
				msg := "(Azure Client) StreamingRecognize: Could not close stream"
				fmt.Printf(msg)
				return
			}

			break
		}

		azure_stream.Write(payload)
	}

	return
}

func azureSessionHandler(event speech.SessionEventArgs) {
	defer event.Close()
}

func azureRecognitionHandler(event speech.SpeechRecognitionEventArgs) {
	defer event.Close()
}
