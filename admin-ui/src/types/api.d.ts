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
  genre: string;
  // YYYY-MM-DD, or empty for a game with no date yet.
  release_date: string;
  kind: GameKind;
  // A YouTube link, or empty. Only YouTube: it is the only player the
  // public site knows how to build.
  video_url: string;
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
  genre: string;
  release_date: string;
  kind: GameKind;
  video_url: string;
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
  genre: string;
  release_date: string;
  kind: GameKind;
  video_url: string;
  is_for_sale: boolean;
  price_display: string;
  // Edited as a real list and serialised to external_links_json on submit;
  // the API stores it as raw JSON text.
  external_links: GameExternalLink[];
  is_published: boolean;
}

// A jam entry is built against a clock; everything else is one of the studio's
// own productions. Mirrors the CHECK on games.kind, so widening this means
// widening the column and giving the public site something to draw.
type GameKind = "production" | "gamejam";

// Mirrors the sortable columns whitelisted by games.sortableColumns in Go.
type GameSortField =
  | "title"
  | "slug"
  | "release_date"
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

interface ResponseAward {
  id: string;
  title: string;
  issuer: string;
  date: string;
  picture_id: string | null;
  game_id: string | null;
  link: string;
  created_at: string;
}

// Create takes no picture or game: both are attached once the award exists.
interface RequestCreateAward {
  title: string;
  issuer: string;
  date: string;
  link: string;
}

interface RequestUpdateAward extends RequestCreateAward {
  picture_id: string | null;
  game_id: string | null;
}

// Mirrors awards.sortableColumns in Go.
type AwardSortField = "title" | "issuer" | "date" | "created_at";

interface AwardSort {
  field?: AwardSortField;
  direction: "asc" | "desc";
}

// picture_id is edit-page state, so it is not a form field. game_id is,
// because it is chosen from a picker rather than uploaded.
interface AwardFormValues {
  title: string;
  issuer: string;
  date: string;
  link: string;
  game_id: string;
}

interface ResponseDevlogPost {
  id: string;
  slug: string;
  title: string;
  content_markdown: string;
  game_id: string | null;
  og_image_id: string | null;
  is_published: boolean;
  // Empty until the post is first published. Stored as YYYY-MM-DD.
  published_at: string;
  created_at: string;
  updated_at: string;
}

// Create takes no game or image: both are attached once the post exists.
interface RequestCreateDevlogPost {
  title: string;
  content_markdown: string;
  is_published: boolean;
  published_at: string;
}

interface RequestUpdateDevlogPost extends RequestCreateDevlogPost {
  game_id: string | null;
  og_image_id: string | null;
}

// Mirrors devlog.sortableColumns in Go.
type DevlogSortField =
  | "title"
  | "is_published"
  | "published_at"
  | "created_at"
  | "updated_at";

interface DevlogSort {
  field?: DevlogSortField;
  direction: "asc" | "desc";
}

// og_image_id is edit-page state, so it is not a form field. game_id is,
// because it is chosen from a picker rather than uploaded.
interface DevlogFormValues {
  title: string;
  content_markdown: string;
  game_id: string;
  is_published: boolean;
  published_at: string;
}

interface ResponseContactSubmission {
  id: string;
  // Empty for anything sent through the public form, which does not ask for a
  // name -- only imported submissions carry one.
  name: string;
  subject: string;
  phone: string;
  email: string;
  message: string;
  wants_callback: boolean;
  is_read: boolean;
  ip_address: string;
  created_at: string;
}

// The unread count travels with the list so the UI never has to derive it
// from a list it may have re-sorted.
interface ResponseContactList {
  submissions: ResponseContactSubmission[];
  unread: number;
}

// Mirrors contact.sortableColumns in Go.
type ContactSortField = "subject" | "email" | "is_read" | "created_at";

interface ContactSort {
  field?: ContactSortField;
  direction: "asc" | "desc";
}

// Mirrors settings.Kind in Go. The kind decides both validation and which
// control the admin gets.
type SettingKind = "text" | "uri" | "uri_list" | "media";

interface SettingDefinition {
  key: string;
  kind: SettingKind;
  multiline: boolean;
  // Only present for media settings: the imaging target the upload goes
  // through, which decides the stored dimensions.
  target?: MediaTarget;
}

// The definitions travel with the values so the form is built from the
// server's registry rather than a second copy of the key list.
interface ResponseSettingsGroup {
  group: string;
  definitions: SettingDefinition[];
  values: Record<string, string>;
}

interface RequestUpdateSettings {
  values: Record<string, string>;
}

// A uri_list setting is edited as a list and serialised to JSON on submit, so
// the form works in a wider shape than the API does.
type SettingsFormValues = Record<string, string | string[]>;

type SettingsGroupName = "site" | "homepage";

interface MediaUsage {
  module: string;
  label: string;
}

interface ResponseMediaItem {
  id: string;
  url: string;
  width: number;
  height: number;
  format: string;
  alt_text: string;
  created_at: string;
  usages: MediaUsage[];
}

// The orphan count travels with the list because picking it out of the grid by
// eye is the tedious part.
interface ResponseMediaLibrary {
  items: ResponseMediaItem[];
  orphaned: number;
}

// Dashboard counters. The shape mirrors internal/stats.Stats; the Go tests pin
// these JSON keys so a rename cannot silently zero the dashboard.
interface ResponseStats {
  games: {
    total: number;
    published: number;
    playable: number;
    for_sale: number;
  };
  devlog: {
    total: number;
    published: number;
  };
  awards: number;
  members: number;
  contact: {
    total: number;
    unread: number;
  };
  media: number;
}
