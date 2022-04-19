"""
Sample usage script for Azure Python SDK
"""
import json
import wave

import azure.cognitiveservices.speech as speechsdk

AZURE_KEY = ""
CHUNK_SIZE = 8000 # 8000 -> 0.5s of audio


def recognize(audio_path):
    print("\n\nRecognizeOnce (SDK): ")
    speech_config = speechsdk.SpeechConfig(
        subscription=AZURE_KEY, region="centralindia", speech_recognition_language="en-IN"
    )
    speech_config.output_format = speechsdk.OutputFormat.Detailed

    # performs one-shot speech recognition with input from an audio file
    audio_config = speechsdk.audio.AudioConfig(filename=audio_path)
    
    # Creates a speech recognizer using a file as audio input, also specify the speech language
    speech_recognizer = speechsdk.SpeechRecognizer(
        speech_config=speech_config, audio_config=audio_config
    )

    response = speech_recognizer.recognize_once()

    if response:
        try:
            response = json.loads(response.json)
        except Exception as e:
            print(e)
            response = []

    print(response)
    result = response.get("NBest", [])


def streaming_recognize(audio_path):
    """performs continuous speech recognition with input from an audio file"""

    speech_config = speechsdk.SpeechConfig(
        subscription=AZURE_KEY, region="centralindia", speech_recognition_language="en-IN"
    )

    speech_config.output_format = speechsdk.OutputFormat.Detailed

    stream = speechsdk.audio.PushAudioInputStream()

    audio_config = speechsdk.audio.AudioConfig(stream=stream)

    speech_recognizer = speechsdk.SpeechRecognizer(
        speech_config=speech_config, audio_config=audio_config
    )

    done = False

    results = []

    def handle_final_result(evt):
        print(evt)
        print(evt.result)
        print(evt.result.txt)
        results.append(evt.result.text)

    def stop_cb(evt):
        """callback that signals to stop continuous recognition upon receiving an event `evt`"""
        print("CLOSING on {}".format(evt))
        nonlocal done
        done = True

    # Connect callbacks to the events fired by the speech recognizer
    speech_recognizer.recognized.connect(handle_final_result)

    speech_recognizer.recognizing.connect(
        lambda evt: print("RECOGNIZING: {}".format(evt))
    )
    speech_recognizer.recognized.connect(
        lambda evt: print("RECOGNIZED: {}".format(evt))
    )
    speech_recognizer.session_started.connect(
        lambda evt: print("SESSION STARTED: {}".format(evt))
    )
    speech_recognizer.session_stopped.connect(
        lambda evt: print("SESSION STOPPED {}".format(evt))
    )
    speech_recognizer.canceled.connect(lambda evt: print("CANCELED {}".format(evt)))
    # stop continuous recognition on either session stopped or canceled events
    speech_recognizer.session_stopped.connect(stop_cb)
    speech_recognizer.canceled.connect(stop_cb)

    # Start continuous speech recognition
    speech_recognizer.start_continuous_recognition()

    n_bytes = 10000
    wav_fh = wave.open(audio_path)

    try:
        while True:
            frames = wav_fh.readframes(n_bytes // 2)
            # print('read {} bytes'.format(len(frames)))
            if not frames:
                break

            stream.write(frames)
    finally:
        # stop recognition and clean up
        wav_fh.close()
        stream.close()
        speech_recognizer.stop_continuous_recognition()

    return results


if __name__ == "__main__":
    recognize("test.wav")
    streaming_recognize("test.wav")