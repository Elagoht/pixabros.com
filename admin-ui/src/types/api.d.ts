interface RequestLogin {
  username: string;
  password: string;
}

interface ResponseLogin {
  username: string;
}

interface ResponseWhoami {
  username: string;
}

interface RequestChangePassword {
  current_password: string;
  new_password: string;
}

interface ResponseMedia {
  id: string;
  url: string;
  width: number;
  height: number;
}

// The upload endpoint resizes to a fixed size per target, so the target
// decides the stored dimensions -- see internal/imaging/targets.go.
type MediaTarget =
  | "avatar"
  | "cd_cover_art"
  | "cartridge_art"
  | "og_image"
  | "screenshot"
  | "award_picture"
  | "org_logo";

interface ResponseGame {
  id: string;
  slug: string;
  title: string;
  short_description: string;
  full_description: string;
  tags: string;
  // Derived server-side from whether a playable build exists; read-only.
  is_browser_playable: boolean;
  is_for_sale: boolean;
  price_display: string;
  external_links_json: string;
  cartridge_art_id: string | null;
  cd_cover_art_id: string | null;
  og_image_id: string | null;
  web_export_path: string;
  display_order: number;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

// Create takes no artwork: media is attached once the game exists.
interface RequestCreateGame {
  title: string;
  short_description: string;
  full_description: string;
  tags: string;
  is_for_sale: boolean;
  price_display: string;
  external_links_json: string;
  display_order: number;
  is_published: boolean;
}

// PUT is a full replace rather than a patch, so every field travels on
// every save -- including the artwork ids the form itself does not edit.
interface RequestUpdateGame extends RequestCreateGame {
  cartridge_art_id: string | null;
  cd_cover_art_id: string | null;
  og_image_id: string | null;
}

interface ResponseScreenshot {
  id: string;
  game_id: string;
  media_id: string;
  display_order: number;
}

interface RequestAddScreenshot {
  media_id: string;
  display_order: number;
}

// Both reorder endpoints take the complete ordered id list, not a delta.
interface RequestReorder {
  ids: string[];
}

// The shape the Formik game form works in. Two fields the API needs are
// deliberately absent: the artwork ids are edit-page state, and display_order
// is owned by the drag-to-reorder control on the list page -- editing it by
// hand here would give the same value two competing sources of truth.
interface GameExternalLink {
  label: string;
  url: string;
}

interface GameFormValues {
  title: string;
  short_description: string;
  full_description: string;
  tags: string;
  is_for_sale: boolean;
  price_display: string;
  // Edited as a real list and serialised to external_links_json on submit;
  // the API stores it as raw JSON text.
  external_links: GameExternalLink[];
  is_published: boolean;
}

// Mirrors the sortable columns whitelisted by games.sortableColumns in Go.
type GameSortField =
  | "title"
  | "slug"
  | "is_published"
  | "display_order"
  | "created_at"
  | "updated_at";

interface GameSort {
  field?: GameSortField;
  direction: "asc" | "desc";
}

interface ResponseMember {
  id: string;
  name: string;
  avatar_id: string | null;
  tags: string;
  description: string;
  links_json: string;
  display_order: number;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

// Create takes no avatar: media is attached once the member exists.
interface RequestCreateMember {
  name: string;
  tags: string;
  description: string;
  links_json: string;
  display_order: number;
  is_published: boolean;
}

interface RequestUpdateMember extends RequestCreateMember {
  avatar_id: string | null;
}

// Mirrors members.sortableColumns in Go.
type MemberSortField =
  | "name"
  | "is_published"
  | "display_order"
  | "created_at"
  | "updated_at";

interface MemberSort {
  field?: MemberSortField;
  direction: "asc" | "desc";
}

// display_order is owned by the drag-to-reorder control on the list page, and
// the avatar id is edit-page state, so neither is a form field.
interface MemberFormValues {
  name: string;
  tags: string;
  description: string;
  links: GameExternalLink[];
  is_published: boolean;
}
