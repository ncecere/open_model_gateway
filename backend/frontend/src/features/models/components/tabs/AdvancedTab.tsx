import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import type { CustomMetadataEntry, ModelFormState } from "../../types";
import { normalizeProviderSlug } from "@/api/model-catalog";
import { DEFAULT_PROVIDER_DETAIL, PROVIDER_DETAILS } from "../../providers";
import { ModelMetadataEditor } from "../ModelMetadataEditor";

const AUDIO_FORMAT_OPTIONS = [
  { value: "json", label: "JSON" },
  { value: "text", label: "Text" },
  { value: "srt", label: "SRT" },
  { value: "vtt", label: "VTT" },
  { value: "verbose_json", label: "Verbose JSON" },
  { value: "diarized_json", label: "Diarized JSON" },
] as const;

const AUDIO_GRANULARITY_OPTIONS = [
  { value: "word", label: "Word-level" },
  { value: "segment", label: "Segment-level" },
] as const;

function parseCSVList(value?: string) {
  if (!value) {
    return [];
  }
  return value
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function formatCSVList(values: string[]) {
  if (values.length === 0) {
    return "";
  }
  return values.join(",");
}

interface AdvancedTabProps {
  form: ModelFormState;
  onChange: (form: ModelFormState) => void;
}

export function AdvancedTab({ form, onChange }: AdvancedTabProps) {
  const providerKey = normalizeProviderSlug(form.provider);
  const providerDetail =
    PROVIDER_DETAILS[providerKey] ?? DEFAULT_PROVIDER_DETAIL;

  const modelType = form.model_type?.toLowerCase();
  const isAudioTranscription = modelType === "audio_transcription";
  const isAudioSpeech = modelType === "audio_speech";

  const handleNumericChange = (key: keyof ModelFormState, value: string) => {
    if (value === "") {
      onChange({ ...form, [key]: "" });
      return;
    }
    const parsed = Number(value);
    if (!Number.isNaN(parsed)) {
      onChange({ ...form, [key]: parsed });
    }
  };

  const handleMetadataValueChange = (key: string, value: string) => {
    const next = { ...form.metadata };
    if (value === "") {
      delete next[key];
    } else {
      next[key] = value;
    }
    onChange({ ...form, metadata: next });
  };

  const handleCustomMetadataChange = (next: CustomMetadataEntry[]) => {
    onChange({ ...form, customMetadata: next });
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="context_window">Context window</Label>
          <Input
            id="context_window"
            value={form.context_window}
            onChange={(event) =>
              handleNumericChange("context_window", event.target.value)
            }
            placeholder="128000"
          />
          <p className="text-xs text-muted-foreground">
            Maximum input token count.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="max_output_tokens">Max output tokens</Label>
          <Input
            id="max_output_tokens"
            value={form.max_output_tokens}
            onChange={(event) =>
              handleNumericChange("max_output_tokens", event.target.value)
            }
            placeholder="4096"
          />
          <p className="text-xs text-muted-foreground">
            Maximum output token count.
          </p>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="weight">Routing weight</Label>
        <Input
          id="weight"
          value={form.weight}
          onChange={(event) =>
            handleNumericChange("weight", event.target.value)
          }
          placeholder="100"
          className="max-w-xs"
        />
        <p className="text-xs text-muted-foreground">
          Higher weights increase the chance of this model being selected during routing.
        </p>
      </div>

      {isAudioTranscription && (
        <AudioTranscriptionSettings
          metadata={form.metadata}
          onMetadataChange={handleMetadataValueChange}
        />
      )}

      {isAudioSpeech && (
        <AudioSpeechSettings
          metadata={form.metadata}
          onMetadataChange={handleMetadataValueChange}
        />
      )}

      <ModelMetadataEditor
        providerDetail={providerDetail}
        metadata={form.metadata}
        customMetadata={form.customMetadata}
        onMetadataChange={handleMetadataValueChange}
        onCustomMetadataChange={handleCustomMetadataChange}
      />
    </div>
  );
}

function AudioTranscriptionSettings({
  metadata,
  onMetadataChange,
}: {
  metadata: Record<string, string>;
  onMetadataChange: (key: string, value: string) => void;
}) {
  const formats = parseCSVList(metadata["audio_formats"]);
  const granularities = parseCSVList(metadata["audio_timestamp_granularities"]);

  const toggleValue = (
    values: string[],
    option: string,
    checked: boolean,
    key: "audio_formats" | "audio_timestamp_granularities",
  ) => {
    const next = new Set(values);
    if (checked) {
      next.add(option);
    } else {
      next.delete(option);
    }
    onMetadataChange(key, formatCSVList(Array.from(next)));
  };

  return (
    <div className="space-y-4 rounded-md border p-4">
      <div>
        <p className="text-sm font-medium">Audio routing settings</p>
        <p className="text-xs text-muted-foreground">
          Configure the response formats and timestamp granularities this model
          can emit.
        </p>
      </div>

      <div className="space-y-2">
        <Label>Allowed response formats</Label>
        <div className="grid gap-2 sm:grid-cols-2">
          {AUDIO_FORMAT_OPTIONS.map((option) => (
            <label
              key={option.value}
              className="flex items-center gap-2 text-sm font-normal"
            >
              <Checkbox
                checked={formats.includes(option.value)}
                onCheckedChange={(checked) =>
                  toggleValue(
                    formats,
                    option.value,
                    Boolean(checked),
                    "audio_formats",
                  )
                }
              />
              {option.label}
            </label>
          ))}
        </div>
      </div>

      <div className="space-y-2">
        <Label>Timestamp granularities</Label>
        <div className="flex flex-wrap gap-3">
          {AUDIO_GRANULARITY_OPTIONS.map((option) => (
            <label
              key={option.value}
              className="flex items-center gap-2 text-sm font-normal"
            >
              <Checkbox
                checked={granularities.includes(option.value)}
                onCheckedChange={(checked) =>
                  toggleValue(
                    granularities,
                    option.value,
                    Boolean(checked),
                    "audio_timestamp_granularities",
                  )
                }
              />
              {option.label}
            </label>
          ))}
        </div>
      </div>
    </div>
  );
}

function AudioSpeechSettings({
  metadata,
  onMetadataChange,
}: {
  metadata: Record<string, string>;
  onMetadataChange: (key: string, value: string) => void;
}) {
  return (
    <div className="space-y-4 rounded-md border p-4">
      <div>
        <p className="text-sm font-medium">TTS defaults</p>
        <p className="text-xs text-muted-foreground">
          Configure the optional fallback voice and format for
          `/v1/audio/speech`.
        </p>
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="audio_voice">Preferred voice</Label>
          <Input
            id="audio_voice"
            value={metadata["audio_voice"] ?? ""}
            onChange={(event) =>
              onMetadataChange("audio_voice", event.target.value)
            }
            placeholder="alloy"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="audio_default_voice">Fallback voice</Label>
          <Input
            id="audio_default_voice"
            value={metadata["audio_default_voice"] ?? ""}
            onChange={(event) =>
              onMetadataChange("audio_default_voice", event.target.value)
            }
            placeholder="verse"
          />
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="audio_format">Audio format</Label>
        <Input
          id="audio_format"
          value={metadata["audio_format"] ?? ""}
          onChange={(event) =>
            onMetadataChange("audio_format", event.target.value)
          }
          placeholder="mp3"
          className="max-w-xs"
        />
      </div>
    </div>
  );
}
