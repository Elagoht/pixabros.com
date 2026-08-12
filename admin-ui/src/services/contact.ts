import { Http } from "@/utilities/http";

// Submissions arrive from the public contact form, so there is no create or
// update here: the admin can only read them, mark them, and delete them.
export class ContactService {
  static list = (sort?: ContactSort): Promise<ResponseContactList> => {
    const query = sort?.field
      ? `?sort=${encodeURIComponent(sort.field)}&dir=${sort.direction}`
      : "";
    return Http.get<ResponseContactList>(`/api/admin/contact${query}`, {
      silent: true,
    });
  };

  static get = (submissionId: string): Promise<ResponseContactSubmission> =>
    Http.get<ResponseContactSubmission>(`/api/admin/contact/${submissionId}`, {
      silent: true,
    });

  // Read state is the only thing the admin can change about a submission.
  static setRead = (
    submissionId: string,
    isRead: boolean,
  ): Promise<ResponseContactSubmission> =>
    Http.put<ResponseContactSubmission>(
      `/api/admin/contact/${submissionId}/read`,
      { is_read: isRead },
      { silent: true },
    );

  static delete = (submissionId: string): Promise<void> =>
    Http.delete<void>(`/api/admin/contact/${submissionId}`, { silent: true });
}
