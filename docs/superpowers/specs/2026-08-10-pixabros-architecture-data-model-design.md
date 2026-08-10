# Pixabros.com — Mimari & Veri Modeli Tasarımı

**Tarih:** 2026-08-10
**Faz:** 1/N — Mimari & Veri Modeli (sonraki fazlar: her modülün implementasyonu için ayrı spec/plan)

## Amaç ve Kapsam

Pixabros.com, Pixabros oyun stüdyosunun tanıtım sitesi + kendi web-export oyunlarını barındıran/serve eden bir platform. Bu doküman, tüm modüllerin üzerine inşa edileceği ortak mimariyi ve veri modelini tanımlar. Modüllerin kendi sayfa davranışları (Landing, Play, Devlog, Ödüller, İletişim) sonraki fazlarda ayrı spec/plan olarak detaylandırılacak.

Kapsam dışı: Ödeme/checkout akışı, RBAC (sadece 2 sabit admin, tam yetki), çoklu kullanıcı yönetimi.

## Genel Mimari

**Backend:** Saf Go (framework yok), `net/http` + Go 1.22+ `http.ServeMux` pattern routing. `modernc.org/sqlite` (CGO-free) ile SQLite.

**Tek origin routing:**
- `/api/*` — JSON API (admin panel + iletişim formu)
- `/I-am-a-pixabro/*` — admin SPA build çıktısı + `/api/admin/*` altındaki admin API'leri
- `/play/{slug}/*` — extract edilmiş oyun web export dosyaları (statik servis)
- `/*` — önceden render edilmiş public HTML sayfaları

**Admin panel:** React + Vite SPA, Service Worker ile app-shell caching (asset'ler cache-first, `/api/*` her zaman network).

**Public site:** Go `html/template` ile sunucu tarafında önceden render edilmiş statik HTML (MPA). Public sayfa içerikleri (HTML/CSS/JS) minify edilir. CSS/JS dosya adları content-hash içerir → `Cache-Control: immutable, max-age=31536000`. HTML dosyaları `ETag` (content hash) + `If-None-Match` → `304` desteği; `Cache-Control: no-cache` (Cloudflare her istekte origin'e doğrular, 304 küçük yanıt döner). Generic 404 sayfası public frontend'in tasarım diliyle (header/footer dahil) hazırlanır, build-time'da statik üretilir.

**Veritabanı erişimi:** `database/sql` + el yazımı SQL, embed edilmiş `.sql` migration dosyaları + `schema_migrations` versiyon tablosu.

**Depolama:** `Storage` arayüzü (`Put`, `Get`, `Delete`, `URL`) + `LocalDiskStorage` implementasyonu (`/data/media/...`, `/data/games/{slug}/...`). İleride S3/R2 gibi implementasyonlar eklenebilir.

## Veri Modeli

### Admin & Auth
```
admins (id, username, password_hash, created_at)
sessions (id, admin_id FK, token_hash, created_at, expires_at)
```
Admin hesapları CLI ile bir kez seed edilir (2 sabit hesap, kullanıcı yönetim ekranı yok). Admin panelde sadece "parolamı değiştir" ekranı vardır.

### Media (ortak medya tablosu)
```
media (id, path, width, height, format='webp', alt_text, created_at)
```
Tüm modüllerin görselleri buraya FK ile referans verir. Upload sırasında orijinal dosya resize + WebP encode edildikten sonra atılır (orijinal saklanmaz).

**Sabit görsel boyutları:**

| Görsel | Boyut | Not |
|---|---|---|
| avatar (Members) | 400×400 | kare |
| cd_cover_art (Games) | 600×600 | kare, portfolio slider'da da kullanılır |
| cartridge_art (Games) | 400×560 | dikey ~5:7, sadece `is_browser_playable=true` ise kullanılır |
| og_image (games/devlog/site default) | 1200×630 | standart OG oranı |
| screenshots (game gallery) | 1280×720 | 16:9 |
| awards picture | 320×320 | kare rozet |
| org_logo (site_settings, JSON-LD) | 512×512 | kare |

### Site Settings (KV, global meta/SEO varsayılanları)
```
site_settings (key TEXT PK, value TEXT, value_type ENUM('text','uri'))
-- örn: site_name, default_og_image (media id), twitter_handle,
--      org_logo (media id), org_sameas_json (sosyal profil linkleri)
```

### Homepage (KV, anasayfaya özel içerik)
```
homepage_settings (key TEXT PK, value TEXT, value_type ENUM('text','uri'))
-- örn: hero_logo, hero_slogan, hero_description, hero_cta_text, hero_cta_link,
--      members_section_title, members_section_subtitle,
--      sales_section_title, portfolio_section_title
```

### Members
```
members (id, name, avatar_id FK media, tags TEXT (virgüllü),
         description, links_json TEXT (JSON [{label,url,icon}]),
         display_order INT, is_published BOOL, created_at, updated_at)
```

### Games
```
games (id, slug (otomatik üretilir, immutable, unique), title,
       short_description, full_description, tags TEXT (virgüllü),
       is_browser_playable BOOL, is_downloadable BOOL, is_for_sale BOOL,
       price_display TEXT nullable (salt gösterim, checkout yok),
       external_links_json TEXT (JSON [{label,url,icon}], örn. Steam/itch),
       cartridge_art_id FK media nullable,  -- sadece browser playable ise anlamlı
       cd_cover_art_id FK media,            -- her oyunda var, portfolio slider'da kullanılır
       og_image_id FK media,
       web_export_path TEXT nullable,       -- extract edilmiş klasör
       display_order INT, is_published BOOL, created_at, updated_at)

game_screenshots (id, game_id FK, media_id FK, display_order INT)
```

### Devlog
```
devlog_posts (id, slug (otomatik, immutable, unique), title,
              content_markdown TEXT, game_id FK nullable,
              og_image_id FK media nullable (boşsa template+başlıktan otomatik üretilir),
              is_published BOOL, published_at, created_at, updated_at)
```

### Awards
```
awards (id, title, issuer, date, picture_id FK media,
        game_id FK nullable, link TEXT nullable, created_at)
```
Sıralama: `date DESC`. Ayrı bir sayfası yok, tek bir listeleme sayfasında gösterilir.

### Contact
```
contact_submissions (id, subject, phone TEXT nullable, email TEXT nullable,
                      message TEXT (min 100 karakter, backend validasyonu),
                      wants_callback BOOL, is_read BOOL default false,
                      ip_address TEXT, created_at)
```
`wants_callback=true` ise backend `phone` veya `email`'den en az birini zorunlu kılar.

### Cache / Dependency Graph & Regen Kuyruğu
```
rendered_pages (page_key TEXT PK, etag TEXT, rendered_at)
page_tags (page_key FK, tag TEXT, PRIMARY KEY(page_key, tag))
-- örn tag'ler: 'game:42', 'game:list', 'homepage', 'devlog:list', 'site_settings'

regen_jobs (id, tag TEXT, status ENUM('pending','processing','done','failed'),
            created_at, processed_at, error TEXT nullable)
```

## Render & Cache Pipeline

1. Admin bir kaydı kaydeder → backend DB'yi günceller → ilgili tag(ler) için `regen_jobs`'a satır eklenir (örn. oyun güncellenince `game:{id}` ve `game:list`).
2. Arka planda çalışan tek bir worker goroutine (polling, 1-2 saniyede bir), `pending` job'ları çeker, ilgili tag'e sahip `page_tags` kayıtlarını bulur.
3. Her bağımlı sayfa yeniden render edilir (`html/template`), içerik hash'i hesaplanır, `/data/rendered/{page_key}.html`'e yazılır, `rendered_pages.etag` güncellenir. Render sırasında sayfanın bağımlı olduğu tag seti (`page_tags`) de güncellenir.
4. Job `done` işaretlenir; hata olursa `failed` + `error`, otomatik retry yapılmaz — admin panelden manuel retry tetiklenir.

**HTTP serving:** Handler, istenen path için `rendered_pages.etag`'i okur, `If-None-Match` ile karşılaştırır → eşleşirse `304`. CSS/JS content-hash'li dosyalar hiç ETag kontrolüne girmeden `immutable` cache header'ı alır.

## Görsel & Medya Pipeline

**Upload akışı:**
1. Admin görsel yükler → backend decode eder → hedef alana göre sabit boyuta crop-to-fill resize eder (merkez odaklı) → pure-Go bir WebP encoder ile WebP'ye encode eder (CGO'suz; kütüphane seçimi implementasyon aşamasında netleşir ve doğrulanır).
2. `/data/media/{yyyy}/{random-id}.webp`'e yazılır, `media` tablosuna satır eklenir. Orijinal dosya atılır.

**Oyun web export upload:**
1. Admin `.zip`, `.tar` veya `.tar.gz`/`.tgz` yükler.
2. Backend arşivi `/data/games/{slug}/build/` altına extract eder (path traversal koruması ile), kök dizinde `index.html` yoksa hata döner ve extract iptal edilir.
3. Başarılıysa `games.web_export_path` güncellenir, `game:{id}` tag'i için regen job'u tetiklenir.
4. `/play/{slug}/*` istekleri doğrudan bu klasörden statik servis edilir.

**Devlog OG resmi:** Sabit bir arka plan template (1200×630) üzerine gömülü font ile başlık çizilir, WebP olarak `media`'ya kaydedilir, `devlog_posts.og_image_id` bu ID'ye işaret eder (admin override yüklerse ID değişir).

**Orphan medya süpürme:** Günlük çalışan bir goroutine, tüm tablolardaki (`games`, `members`, `devlog_posts`, `awards`, `game_screenshots`, `site_settings`, `homepage_settings`) medya referanslarını toplar; referans edilmeyen ve `created_at`'i 24 saatten eski olan `media` kayıtlarını + dosyalarını siler (24 saatlik gecikme, yarım kalan upload'ları korumak için).

## Admin Panel & Auth

- `POST /api/admin/login` — bcrypt kontrolü, başarılıysa `sessions`'a satır + `HttpOnly, Secure, SameSite=Strict` cookie (ham token cookie'de, hash DB'de). 7 gün sliding expiration.
- `POST /api/admin/logout` — session satırı silinir.
- `POST /api/admin/change-password` — admin kendi parolasını değiştirir.
- Admin SPA: her modül için CRUD ekranları (Games, Members, Devlog, Awards, Contact inbox read+is_read toggle, Homepage/Site Settings KV editörü, Media kütüphanesi görüntüleme), regen job durumu ekranı (pending/failed job'ları görme + manuel retry).

**İletişim formu spam koruması:** Honeypot alan (doluysa sessizce 200 dön) + IP başına basit in-memory rate limit (örn. 1 dakikada 1 istek).

## Hata Yönetimi

- API'de tutarlı JSON hata formatı: `{"error": {"code": "...", "message": "..."}}`, uygun HTTP status kodları (400/401/404/409/500).
- Validasyon backend'de zorunlu (frontend validasyonu sadece UX).
- Regen job hataları admin panelde görünür kalır, otomatik retry yok.
- Bozuk arşiv/path traversal/eksik `index.html` durumunda extract tamamen geri alınır, `web_export_path` güncellenmez.
- Public 404 sayfası build-time'da bir kez üretilen statik bir sayfa, render pipeline'ından bağımsız.

## Test Stratejisi

- Backend: Go `testing` paketi, DB katmanı için gerçek SQLite (in-memory/temp dosya) ile entegrasyon testleri, mock yok.
- Render pipeline: "içerik değişti → doğru tag'ler işaretlendi → doğru sayfalar regen edildi" senaryoları.
- Görsel pipeline: resize+WebP encode çıktısının boyut/format doğruluğu için unit testler.
- Admin SPA: kritik CRUD akışları için Vitest + React Testing Library; auth flow (login/logout/session expiry) ayrıca test edilir.
- Public sayfa render'ları için golden-file (snapshot) testleri.

## Sonraki Fazlar

Bu spec onaylandıktan sonra her modül (Anasayfa, Play Sayfası, Devlog, Ödüller, İletişim) için ayrı implementasyon planı çıkarılacak; bu doküman tümüne temel oluşturur.
