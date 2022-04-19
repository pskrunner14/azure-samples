"""
Sample usage script for Azure Speech to Text REST API
"""
import json
import requests

AZURE_KEY = ""
CHUNK_SIZE = 8000 # 8000 -> 0.5s of audio

def recognize(audio_path):
    print("\n\nRecognize (REST): ")
    recognize_url = "https://centralindia.stt.speech.microsoft.com/speech/recognition/conversation/cognitiveservices/v1?language=en-IN&format=detailed"
    headers = {
        "Ocp-Apim-Subscription-Key": AZURE_KEY,
        "Content-Type": "audio/wav; codecs=audio/pcm; samplerate=8000"
    }

    with open(audio_path, "rb") as f:
        audio_bytes = f.read()

    response = requests.post(recognize_url, data=audio_bytes, headers=headers)
    print(json.loads(response.text).get("NBest", []))


def streaming_recognize(audio_path):
    print("\n\nStreaming Recognize (REST): ")
    recognize_url = "https://centralindia.stt.speech.microsoft.com/speech/recognition/conversation/cognitiveservices/v1?language=en-IN&format=detailed"
    headers = {
        "Ocp-Apim-Subscription-Key": AZURE_KEY,
        "Content-Type": "audio/wav; codecs=audio/pcm; samplerate=8000",
        "Transfer-Encoding": "chunked"
    }

    with open(audio_path, "rb") as f:
        audio_bytes = f.read()

    print(len(audio_bytes))

    def audio_from_source(audio):
        for i in range(0, len(audio), CHUNK_SIZE):
            yield audio[i: i + CHUNK_SIZE]

    response = requests.post(recognize_url, data=audio_from_source(audio_bytes), headers=headers, stream=True)
    for resp in response.iter_lines():
        print(resp)


if __name__ == "__main__":
    recognize("test.wav")
    streaming_recognize("test.wav")