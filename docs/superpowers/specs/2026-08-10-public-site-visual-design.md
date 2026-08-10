# Pixabros.com — Public Site Görsel Tasarım Kararları

**Tarih:** 2026-08-10

## Amaç

`2026-08-10-frontend-design-and-stack.md` genel renk paleti/tipografi/stack kararlarını netleştirmişti. Bu doküman, public site'ın (statik MPA) her sayfası/bölümü için somut görsel/etkileşim kararlarını kaydeder — brainstorming'in görsel companion aracıyla mockup'lar üzerinden onaylandı. Bu doküman, public site'ın implementasyon planı yazılırken temel referans olacak.

Renk paleti, tipografi ve genel stack `2026-08-10-frontend-design-and-stack.md`'de sabit — burada tekrar edilmiyor.

## Landing Page

### Hero Bölümü
**Bölünmüş (Split) düzen.** Sol: logo + slogan + açıklama + CTA butonu, sol hizalı. Sağ: öne çıkan görsel (oyun görseli veya portfolyo önizlemesi). Steam mağaza sayfası hissine yakın.

### Portfolio Slider
Steam anasayfasının "Öne Çıkanlar ve Tavsiye Edilenler" carousel'i referans alınır:
- Tam genişlik carousel, kenarlarda önceki/sonraki kartın kısmi (soluk) görünümü.
- Sol ~65%: büyük görsel alanı, varsayılan olarak oyunun `cd_cover_art`'ı gösterilir.
- Sağ ~35%: oyun başlığı, tag'ler (pill/etiket olarak), ve oyunun screenshot'larından oluşan 2x2 küçük thumbnail grid'i.
- **Etkileşim:** Bir screenshot thumbnail'ine hover edildiğinde büyük görsel alanı o screenshot'a döner; hover kalkınca `cd_cover_art`'a geri döner.
- Alt/üstte ok (‹ ›) ile slide geçişi + altta nokta (dot) pagination.
- Karta veya büyük görsele tıklanınca oyunun detay sayfasına gidilir.

### Satıştaki Oyunlar Grid'i
**Bindirmeli kart (overlay):** `cd_cover_art` kartın tamamını kaplar, altından koyu gradient ile başlık + tag pill'leri görsel üzerine bindirilir. Kare format korunur, kompakt.

### Members (Üyeler) Bölümü
**Dikey kart, yuvarlak avatar + bio.** Sırasıyla: yuvarlak avatar (üstte, vurgu renginde ince çerçeve), isim, rol/unvan, mini bio paragrafı (description alanından), tag pill'leri (comma-separated tags alanından). Her şey tek sütun, ortalı.

## Play Sayfası

### TV + Konsol Alanı
**Tam skeuomorfik.** Gerçekçi bir CRT TV illüstrasyonu (yuvarlatılmış köşeler, koyu bezel) içinde oyunun canvas'ı/iframe'i gösterilir. Altında ahşap dokulu bir NES konsolu illüstrasyonu (kontrolcü şekilleri dekoratif olarak durabilir). Bu, sitenin GENEL sade/modern dilinden bilinçli olarak ayrılan, sadece bu sayfaya özel retro bir "sahne".

### Kartuş Grid'i (Browser Playable Oyunlar)
Gerçek bir NES kartuşuna birebir referans alınır (kullanıcının verdiği fotoğraf): gri plastik gövde, solda oluklu tutma/parmak izi bölgesi, sağda/merkezde etiket alanı — `cartridge_art` (400×560) bu etiket alanına tam oturur, altta küçük üçgen çentik dekoratif detay olarak eklenebilir. Grid'de bir satırda birden fazla kartuş yan yana durur, hover'da hafif büyüme/kalkma efekti.

### CD Kutuları Grid'i (Downloadable Oyunlar)
Kare CD kutusu kartları, bir satırda en fazla 2 adet (spec'te zaten sabitti). **Tıklanınca 3D kapak açılışı:** kutu gerçek bir menteşeden açılır gibi perspective flip animasyonuyla döner, içinden oyun detayları (açıklama, tag'ler, screenshot galerisi, harici linkler) bir panel olarak ortaya çıkar.

## Devlog

**Kompakt liste satırı.** Küçük thumbnail (OG görseli/kapak) solda, başlık + tarih + (varsa) ilişkili oyun adı sağda — klasik blog listesi görünümü, dikey liste, hızlı taranabilir. Oyuna göre filtrelenebilir (dropdown/pill filtre — implementasyon planında detaylanacak).

## Ödüller

**Kronolojik zaman çizelgesi (timeline).** Dikey bir çizgi üzerinde tarihe göre sıralı (en yeni üstte), her ödül küçük bir rozet görseli (`picture_id`, 320×320) + başlık/kurum/tarih ile satır olarak durur. "Yolculuk" hissi veren bir vitrin.

## İletişim

**Tek sütun, ortalanmış kart.** Form tek başına sayfa ortasında bir kart içinde — ekstra yan panel/bilgi bloğu yok. Alanlar spec'teki sırayla: konu, telefon (opsiyonel), eposta (opsiyonel), mesaj (min 100 karakter), "geri dönüş istiyorum" checkbox'ı, gönder butonu.

## Sonraki Adım

Bu kararlar, public site'ın implementasyon planı yazılırken (henüz yazılmadı — Plan A/B/C'nin üzerine gelecek bir sonraki faz) doğrudan kullanılacak. El yazımı düz CSS ile uygulanacak (bkz. stack spec'i), tasarım token'ları oradaki `:root` custom property bloğundan referans verilecek.
