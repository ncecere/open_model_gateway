# Audio
Handle transcription, translation, and speech synthesis with multipart uploads.

## Prepare environment
Set the API base, key, and audio file paths.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
export AUDIO_FILE_PATH="~/Downloads/sample.wav"
export SPEECH_OUTPUT_PATH="speech.mp3"
```

## Multipart fields
Honor OpenAI's schema when building transcription or translation requests.

| Field | Required | Notes |
| --- | --- | --- |
| `model` | ✅ | Tenant-scoped alias such as `gpt-4o-transcribe`. |
| `file` or `audio` | ✅ | MP3, MP4, MPEG, MPGA, WAV, or WEBM payload. |
| `response_format` | ❌ | `json` (default), `text`, `srt`, `verbose_json`, `vtt`. |
| `timestamp_granularities[]` | ❌ | Accepts `word` or `segment` when `response_format=verbose_json`. |
| `prompt`, `language`, `temperature`, `user` | ❌ | Provider dependent hints. |
| `stream` | ❌ | Requires providers with `audio_streaming=true` metadata. |

## Transcribe audio
Send multipart form data to `/v1/audio/transcriptions` and parse the JSON body.

```bash
curl -sS "$GATEWAY_BASE_URL/audio/transcriptions" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: multipart/form-data" \
  -F "model=gpt-4o-transcribe" \
  -F "file=@${AUDIO_FILE_PATH}" \
  -F "response_format=verbose_json" | jq '{text, segments: .segments[0:2]}'
```

## Translate audio
Swap the endpoint for `/v1/audio/translations` and add `task=translate` if the provider expects it.

```bash
curl -sS "$GATEWAY_BASE_URL/audio/translations" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: multipart/form-data" \
  -F "model=gpt-4o-transcribe" \
  -F "file=@${AUDIO_FILE_PATH}" \
  -F "prompt=Summarize the clip in English" | jq '.text'
```

## Stream transcriptions
Set `stream=true` and `response_format=diarized_json` to receive SSE events, but only models with `audio_streaming=true` metadata accept the upgrade today.

```bash
curl -sN "$GATEWAY_BASE_URL/audio/transcriptions" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Accept: text/event-stream" \
  -F "model=gpt-4o-transcribe" \
  -F "file=@${AUDIO_FILE_PATH}" \
  -F "stream=true" \
  -F "response_format=diarized_json"
```

## Synthesize speech
Send JSON to `/v1/audio/speech`, choose a voice, and write the binary response to disk.

```bash
curl -sS "$GATEWAY_BASE_URL/audio/speech" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-4o-mini-tts",
        "voice": "alloy",
        "input": "Open Model Gateway is routing traffic successfully."
      }' --output "$SPEECH_OUTPUT_PATH"
echo "Saved speech synthesis to $SPEECH_OUTPUT_PATH"
```

## Response formats
`json` responses include `{text, language, duration, segments}` while non-JSON formats stream raw text; usage tokens may only appear in JSON payloads so capture metrics via logs when needed.

## Monitor headers
Budget, rate-limit, request ID, and provider ID headers help reconcile billing even when the body is not JSON.
