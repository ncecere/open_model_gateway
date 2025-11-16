# Audio API Surface (OpenAI Compatibility)

This document summarizes the OpenAI `/v1/audio/*` REST contracts that the gateway must honor so SDKs and integrations behave identically.

## Endpoints

- `POST /v1/audio/transcriptions`
- `POST /v1/audio/translations`
- `POST /v1/audio/speech` (already implemented; listed for completeness)

Both transcription endpoints share the same multipart request grammar and differ only in intent (`task=transcribe` vs. `translate`). When a provider cannot satisfy a requested feature (e.g., timestamp granularities), the request must short-circuit with a structured error rather than falling through to an incompatible adapter.

## Multipart Request Schema

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `model` | string | ✅ | Logical model alias (tenant-scoped). |
| `file` | file upload | ✅ | Primary audio payload (MP3, MP4, MPEG, MPGA, WAV, WEBM). We also accept `audio` as an alias. |
| `response_format` | string | ❌ | Defaults to `json`. Supported values: `json`, `text`, `srt`, `verbose_json`, `vtt`. |
| `temperature` | float | ❌ | 0–1; controls sampling. |
| `language` | string | ❌ | BCP‑47 language code. Translation ignores this value. |
| `prompt` | string | ❌ | Optional text prompt that biases transcription. |
| `timestamp_granularities[]` | array | ❌ | Accepts `word` and/or `segment`. Requires `response_format=verbose_json`. |
| `user` | string | ❌ | Usage/audit metadata; pass-through to upstream providers if supported. |
| `stream` | bool | ❌ | When `true`, the gateway upgrades to SSE and proxies provider events (`response_format` must be `diarized_json`). |

Additional OpenAI fields (`audio`, `initial_prompt`, `temperature`, `compression_ratio_threshold`, etc.) are not GA in their public docs; we will add them once upstream providers surface equivalents.

Streaming is only enabled for providers/models that advertise `audio_streaming=true` in their catalog metadata. Today only the native OpenAI adapter exposes this capability; Azure OpenAI and other providers will return `400 model does not support streaming transcriptions` even if `stream=true` is provided. If a tenant requests streaming against a route that does not opt in, the gateway short-circuits the request with that error instead of proxying it upstream.

## Responses

The handler must emit the exact format dictated by `response_format` while still logging usage records internally:

- `json` (default) → `application/json` body:
  ```json
  {
    "text": "Hello world",
    "language": "english",
    "duration": 1.23,
    "segments": [...]
  }
  ```
- `text` → `text/plain; charset=utf-8` with raw transcript.
- `srt` → `text/plain; charset=utf-8` SRT document.
- `vtt` → `text/vtt; charset=utf-8`.
- `verbose_json` → JSON with metadata per segment and optional timestamp granularities.

Translations follow the same formats but omit `language` and `segments` metadata in OpenAI’s current API behavior.

## Usage Logging

OpenAI includes the following usage block in JSON responses:

```json
{
  "usage": {
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "total_tokens": 1234
  }
}
```

Only the `total_tokens` field is populated today. Non-JSON formats do not embed usage data, so the adapters must capture usage metrics out-of-band (if provided in headers) or default to zero.

## Acceptance Criteria

1. Gateway handlers must parse/validate every field listed above, reject unsupported combinations (e.g., `timestamp_granularities` without `verbose_json`), and enforce the configured max upload size.
2. Providers declare their supported audio formats + timestamp granularities so routing can decline unsupported requests without invoking the adapter.
3. OpenAI + Azure OpenAI adapters must:
   - Send `response_format` and `timestamp_granularities[]` via the SDK or raw multipart form.
   - Return raw response bytes + parsed helpers for the handler.
   - Surface usage metadata for transcriptions and translations.
4. Comprehensive tests cover multipart parsing, format negotiation, and adapter serialization/deserialization paths.

Any fields not yet supported (streaming transcriptions, inline PCM responses) should be tracked as follow-up items in the roadmap/tasks backlog.
