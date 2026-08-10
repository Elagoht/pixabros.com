# Pixabros.com — Frontend Design Language & Stack

**Tarih:** 2026-08-10

## Amaç

Faz 1 (Backend Core, Content Pipelines, Admin SPA Shell) mimari planları hazırlanırken görsel tasarım dili ve frontend teknoloji seçimleri hiç konuşulmamıştı. Bu doküman o boşluğu kapatır: hem admin paneli hem public site için ortak görsel kimlik + admin panelinin kesin kütüphane/stil stack'ini tanımlar. Bu doküman onaylandıktan sonra `docs/superpowers/plans/2026-08-10-admin-spa-shell.md` (Plan C) bu karara göre yeniden yazılır.

## Genel Tasarım Yönü

Modern koyu tema, oyun stüdyosu / dev-studio hissi (Steam/itch.io sayfalarına yakın): sade, enerjik, minimal. Retro unsurlar (NES konsolu, kartuş, CD kutusu, pixel font) **sadece Play sayfasındaki ilgili bölümde** yaşar — sitenin genel dili bunlarla boğulmaz. Admin paneli aynı renk/tipografi dilini kullanır ama tamamen işlevsel/sade kalır, retro unsur hiç barındırmaz.

## Renk Paleti

| Rol | Değer | Kullanım |
|---|---|---|
| Sayfa arka planı | `#0F1115` | body/page background |
| Kart/yüzey arka planı | `#171A21` | card, panel, modal, input background |
| Birincil metin | `#F1F1F3` | başlıklar, ana metin |
| İkincil/soluk metin | `#9AA0AC` | açıklama, placeholder, meta bilgi |
| Kenarlık/ayırıcı | `#2A2E37` | border, divider |
| Vurgu (magenta/mor-pembe) | `#E879F9` | CTA, link, aktif durum, focus ring |
| Vurgu (hover/koyu) | `#C026D3` | vurgu renginin hover/active hali |
| Başarı | `#34D399` | success toast, onay durumları |
| Hata | `#F87171` | error toast, form validasyon hatası |
| Uyarı | `#FBBF24` | uyarı/dikkat durumları |

Bu token'lar admin panelde Tailwind'in `theme.extend.colors` alanında, public site'da ise bir `:root { --color-*: ... }` CSS custom property bloğunda tanımlanır — iki taraf da aynı ham değerlere referans verir, tek bir yerde güncellenir.

## Tipografi

- Genel UI (admin panel + public site, retro bölüm hariç her yer): **Inter** (sans-serif).
- Play sayfasındaki NES/CD retro bölümü: **Press Start 2P** (pixel font) — sadece o bölümde, sitenin başka hiçbir yerinde kullanılmaz.

## Admin Panel Teknoloji Stack'i

- Vite + React + TypeScript (değişmedi)
- `react-router-dom` — routing (değişmedi)
- **Tailwind CSS v3** — utility-first styling, renk paleti `tailwind.config`'te tanımlanır
- **Formik + Yup** — form state yönetimi ve şema validasyonu
- **TanStack Query (React Query)** — API veri çekme, cache, mutation yönetimi (whoami query, login/logout/change-password mutation'ları)
- **sonner** — toast bildirimleri (başarı/hata mesajları)
- **classnames** — koşullu class birleştirme
- **shadcn/ui + Radix primitives** — modal, dropdown, select, input, button gibi erişilebilir component'ler; kod projede yaşar (npm paketi değil, kopyalanan component), tasarım dili tam kontrol edilebilir

## Public Site Teknoloji

- El yazımı düz CSS (Tailwind kullanılmaz) — yukarıdaki renk/tipografi token'larını CSS custom property olarak paylaşır.
- Statik HTML/CSS/JS, content-hash'li immutable cache (mevcut mimari kararı değişmedi, bkz. `2026-08-10-pixabros-architecture-data-model-design.md`).

## Sonraki Adım

`docs/superpowers/plans/2026-08-10-admin-spa-shell.md` bu stack'e göre baştan yazılacak: Tailwind v3 kurulumu + renk token'ları, shadcn/ui component kurulumu, Formik+Yup ile login/parola değiştirme formları, TanStack Query ile whoami/login/logout/change-password, sonner ile başarı/hata bildirimleri.
