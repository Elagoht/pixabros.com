import { Http } from "@/utilities/http";

// The library browses what is already stored. Uploading lives with the module
// that needs the image, because the upload target decides its dimensions.
export class MediaLibraryService {
  static list = (): Promise<ResponseMediaLibrary> =>
    Http.get<ResponseMediaLibrary>("/api/admin/media", { silent: true });

  // Alt text is the one field an upload cannot decide for itself, and it is
  // what a screen reader reads out on the public site.
  static setAltText = (mediaId: string, altText: string): Promise<void> =>
    Http.put<void>(
      `/api/admin/media/${mediaId}`,
      { alt_text: altText },
      { silent: true },
    );

  static delete = (mediaId: string): Promise<void> =>
    Http.delete<void>(`/api/admin/media/${mediaId}`, { silent: true });
}
