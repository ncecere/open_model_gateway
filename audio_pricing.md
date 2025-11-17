# Audio Pricing Notes

This document outlines how we plan to track and bill audio workloads. Nothing below is implemented yet—it simply captures the agreed approach so we can wire it up when the backend is ready.

## Pricing Standards

- **Speech-to-Text (STT)**: Most providers bill per minute of input audio. We’ll mirror that by capturing the input duration and applying a per-minute rate.
- **Text-to-Speech (TTS)**: Commonly billed per million output characters. We’ll count the characters synthesized and apply the per-million rate.

## Catalog Metadata

Each audio-capable alias in the model catalog can declare pricing metadata so the executor knows how to calculate charges:

```yaml
metadata:
  price_audio_minute_cents: "60"             # e.g. $0.60/min for STT
  price_tts_million_chars_cents: "1500"     # e.g. $15/1M chars for TTS
```

Only one of the fields needs to be present depending on the capability:

- STT routes (`model_type: audio_transcription`/`audio_translation`) use `price_audio_minute_cents`.
- TTS routes (`model_type: audio_speech`) use `price_tts_million_chars_cents`.

These fields simply expose the desired price; they do not affect billing until the executor reads them.

## Planned Implementation

1. **Capture metrics**
   - STT: determine input duration (from metadata or by probing the uploaded file) and store it in the usage record.
   - TTS: count the characters in the synthesized response (or the request payload) and store the count in the usage record.
2. **Cost calculation**
   - If `price_audio_minute_cents` is present, convert duration → minutes and calculate cost.
   - If `price_tts_million_chars_cents` is present, convert characters → millions and calculate cost.
   - Fall back to provider-reported usage (tokens) if custom pricing metadata isn’t set.
3. **Usage logging**
   - Extend `usagepipeline.Record` to store audio durations/character counts.
   - Use `OverrideCostCents` so budgets/usage dashboards reflect the derived cost.
4. **UI/Docs**
   - Admin catalog editor exposes the new metadata fields with tooltips (per-minute/per-million hints).
   - Admin/user usage pages show audio durations and character counts where applicable.

## Notes

- These metadata keys already appear in `deploy/router.example.yaml` for illustrative purposes; they’re safe to include in real configs even before the computation logic lands.
- When we implement this, make sure to update the changelog, config docs, and the audio API documentation so operators know how to configure pricing.
