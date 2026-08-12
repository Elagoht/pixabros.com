import { Http } from "@/utilities/http";

export class MediaService {
  static get = (mediaId: string): Promise<ResponseMedia> =>
    Http.get<ResponseMedia>(`/api/admin/media/${mediaId}`, { silent: true });

  // The server resizes to the target's fixed dimensions and stores WebP, so
  // the caller only picks which target the image is for.
  static upload = (file: File, target: MediaTarget): Promise<ResponseMedia> => {
    const body = new FormData();
    body.append("file", file);
    return Http.post<ResponseMedia>(
      `/api/admin/media/upload?target=${target}`,
      body,
      { silent: true },
    );
  };
}
