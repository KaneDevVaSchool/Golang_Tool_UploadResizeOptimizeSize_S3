import type { ArtworkMetaFormValues } from "./ArtworkMetaForm";

export type BulkRowStatus = "pending" | "saving" | "done" | "error";

export type BulkRow = {
  localId: string;
  file: File;
  localPreview: string;
  values: ArtworkMetaFormValues;
  status: BulkRowStatus;
  error?: string;
};
