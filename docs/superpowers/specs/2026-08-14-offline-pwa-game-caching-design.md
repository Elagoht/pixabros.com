# Pixabros.com — Çevrimdışı PWA ve Oyun Cache'leme Tasarımı

**Tarih:** 2026-08-14

## Amaç

Site 2026-08-14'te kurulabilir bir PWA oldu: `manifest.webmanifest`, favicon'lar ve ikonlar var, service worker **yok**. Bu doküman eksik yarıyı, yani çevrimdışı çalışmayı tasarlar.

Çevrimdışı desteğin varlık sebebi tek bir gereksinim: **`/play/*` altındaki oyun build'lerinin de cache'lenmesi.** Sadece sayfaları çevrimdışı açan bir PWA bu proje için değersiz kabul edildi. Bütün kararlar bu kısıttan türüyor.

Boyutlar sorunun kendisi: `data/games` 2026-08-14 itibarıyla 386 MB, tek tek build'ler 39–93 MB (dungrid-tactics 93, har 62, the-forking-path 60). Hiçbir şey habersiz indirilemez.

Mimari/veri modeli kararları `2026-08-10-pixabros-architecture-data-model-design.md`'de, görsel dil `2026-08-10-public-site-visual-design.md`'de sabit — burada tekrar edilmiyor.

## Mevcut Durumun Tespiti

Tasarımı belirleyen dört bulgu:

1. **Godot export'larının kendi service worker'ı ölü doğmuş.** Sekiz oyundan ikisi (tetrabros, magical-veins) PWA seçeneğiyle export edilmiş. İkisinde de worker'ın `CACHED_FILES` listesi `<oyun>.html` diyor ama diskteki dosyanın adı `index.html`; `cache.addAll()` tek bir 404'te reddettiği için install başarısız oluyor ve worker hiç aktive olmuyor. Listede `.pck` ve `.wasm` de yok, yani ağırlığın tamamı zaten kapsam dışı. Sonuç: `/play/{slug}/` kapsamı bugün boşta.
2. **Build'lerin dosya listesi hiçbir yerde tutulmuyor.** `gamearchive.Extract` yalnızca kökte `index.html` var mı diye bakıyor; `/play/` düz bir dosya sunucusu (`noDirListing`). Ama extractor bizim kodumuz.
3. **`games` tablosunda build sürümü yok.** `web_export_path` ve `updated_at` var, ikisi de build'in içeriğini tarif etmiyor.
4. **Arşivlerde macOS çöpü var ve herkese açık servis ediliyor.** tetrabros'un 30 dosyasının 15'i `__MACOSX/._*` kaynak çatalı.

## Kararlar

Brainstorming sırasında onaylanan dört ürün kararı:

| Karar | Seçim |
|---|---|
| Oyun nasıl indirilir | Oyun sayfasında boyutu söyleyen açık bir buton |
| Site kendisi ne kadar çevrimdışı çalışır | Kabuk + gezilen sayfalar |
| Build değişince ne olur | Eski kopya çalışmaya devam eder, "yeni sürüm var" denir, kullanıcı karar verir |
| Mimari | Tek kök service worker (aşağıda A) |

### Neden tek kök worker

- **A — Tek kök worker (seçilen).** `/sw.js`, scope `/`. Site kabuğunu, gezilen sayfaları ve `/play/*`'ı tek yaşam döngüsü altında yönetir. Oyun dosyalarını sayfadaki indirme betiği `game-{slug}-{version}` cache'ine yazar (Cache API'ye yazmak scope'a bağlı değil); worker yalnızca okur.
- **B — Site worker + oyun başına worker.** Extraction her oyun için `/play/{slug}/sw.js` üretir. Kapsam çakışması yapısal olarak imkânsız olur, ama N worker'ın ayrı update/lifecycle yönetimi çıkar ve indirme arayüzü başka scope'taki bir worker'la konuşmak zorunda kalır. B'nin tek gerçek kazancını A, extraction'da worker dosyasını silerek elde ediyor.
- **C — Godot'nun kendi PWA'sını düzelt.** Reddedildi: sitenin ana özelliği export ayarına bağlanır, yalnızca Godot oyunlarında çalışır, ve indirme butonu / ilerleme / boyut / güncelleme kararı gibi kararlaştırılmış davranışların hiçbirini vermez.

## Build Manifesti ve Sürüm

`gamearchive.Extract` her arşiv girdisini zaten dolaşıp diske yazıyor; eklenen tek şey defter tutmak. Yeni dönüş değeri:

- yazdığı her dosyanın **göreli yolu ve baytı**,
- **toplam bayt**,
- bir **sürüm damgası**.

Sürüm, dosyalar akarken hesaplanan içerik hash'lerinin yola göre sıralanmış listesinin `sha256`'sıdır. Yani sürüm build'in içeriğidir: aynı arşiv yeniden yüklenirse sürüm değişmez ve kimse aynı 90 MB'ı ikinci kez indirmez. `updated_at` gibi bir zaman damgası bunu yapamazdı.

`__MACOSX/` ve `._*` girdileri manifestten dışlanır **ve diske hiç yazılmaz.** Bugünkü davranışı değiştiriyor; bilinçli, ayrıca onaylandı. Çevrimdışı kopyaya kaynak çatalı indirmek anlamsız, herkese açık servis etmek de zaten istenmeyen bir şeydi.

**Saklama** `external_links_json` konvansiyonunu izler — `games` tablosuna üç sütun, migration 0021:

- `build_version TEXT NOT NULL DEFAULT ''`
- `build_bytes INTEGER NOT NULL DEFAULT 0`
- `build_files_json TEXT NOT NULL DEFAULT '[]'`

Liste her zaman bütün olarak okunduğu için ayrı bir tabloya değmez. Build silindiğinde üçü birden temizlenir.

**Uç nokta:** `GET /api/games/{slug}/build` — oturumsuz, `apiCSP` altında.

```json
{"version": "a1b2c3d4", "bytes": 46137344, "files": ["index.html", "tetrabros.wasm", "..."]}
```

Yalnızca yayınlanmış (`is_published`) ve tarayıcıda oynanabilir (`is_browser_playable`) oyunlar için cevap verir; build'i olmayan oyun 404. Yollar görelidir, istemci başına `/play/{slug}/` ekler.

## Service Worker

### Konum ve kayıt

`/sw.js` sabit bir rotadan servis edilir — scope `/` için betiğin kökte olması gerekir. `Cache-Control: no-cache` ile gider ve URL'i **hash'lenmez**: worker güncellemesi bayt karşılaştırmasıyla yürür, sabit URL burada doğru olandır (sitenin geri kalanındaki içerik-hash kuralının bilinçli istisnası).

Kayıt, her sayfaya eklenen yeni bir `offline.js` içindedir; aynı dosya indirme arayüzünü de taşır.

### Rota sınıflandırması

Sınıflandırma istek tipinden **önce** URL önekine bakar — oyun iframe'i de bir navigasyondur, ama `/play/` kuralına düşmelidir.

| Alan | Strateji | Gerekçe |
|---|---|---|
| Navigasyonlar (HTML) | Önce ağ → cache → çevrimdışı sayfası | Mimarinin tamamı render-store'un ETag tazeliği üzerine kurulu; cache-first onu delerdi |
| `/assets/build/*` | Önce cache | İçerik hash'li ve değişmez |
| `/media/*` | Önce ağ → cache | Medya anahtarları hash'li **değil**; silinen bir görsel gerçekten kaybolabilmeli |
| `/play/{slug}/*` | Önce cache (kullanıcının elindeki sürümden) → ağ | Çevrimdışı oynanabilirliğin kendisi |
| `/api/*`, `/I-am-a-pixabro/*` | Hiç dokunulmaz | Worker admin paneline veya API'ye asla cevap vermez |

### Kabuk listesi

CSS, fontlar, ikonlar ve betikler içerik-hash'li adlar taşır; worker onları gömemez. `GET /api/shell` güncel URL listesini ve bir sürüm damgasını döner (damga bundle hash'lerinden türer). Worker listeyi `install`'da çeker, `activate`'te ve çevrimiçi geçen her navigasyondan sonra damgayı karşılaştırıp değiştiyse yeniler.

Bu yenileme olmadan şu açık kalıyordu: stylesheet değişir, kullanıcı bir daha çevrimiçi girmez, çevrimdışı açtığında sayfa yeni stylesheet URL'ini ister ve cache'te bulamaz — stilsiz sayfa.

### Cache aileleri ve ömür

| Cache | Ömür |
|---|---|
| `shell-{sürüm}` | Damga değişince düşer |
| `pages` | Sınırlı, ~60 girdi |
| `media` | Sınırlı, ~120 girdi |
| `game-{slug}-{version}` | **Otomatik silinmez** |

`pages` ve `media` sınırlıdır, yoksa sınırsız büyürler. Sınır aşılınca **en eski yazılan girdi** düşer: Cache API `keys()`'i ekleme sırasında döndürdüğü için bu, ek defter tutmadan uygulanabilen tek politikadır. Gerçek LRU, her okumada bir zaman damgası yazmayı gerektirirdi ve bu iki cache için o maliyete değmez. Oyun cache'leri asla otomatik düşmez: karar gereği eski kopya çalışmaya devam eder; yalnızca kullanıcının "kaldır"ı veya bilerek yaptığı güncelleme siler. `activate` bu dört aileden olmayan her cache'i temizler.

### Çevrimdışı sayfası

Gerçek bir sayfadır: `internal/site`'a `offline` anahtarıyla girer, 404 gövdesi gibi render edilir ve kabukla birlikte precache'lenir.

### Godot worker'larının etkisizleştirilmesi

Extraction artık `*.service.worker.js` dosyalarını siler, böylece `/play/{slug}/` kapsamı güvenilir biçimde kök worker'ındır.

**Bedeli açıkça kaydediliyor:** Godot'nun worker'ı aynı zamanda cross-origin isolation başlıklarını (COOP/COEP) enjekte ediyordu. İleride SharedArrayBuffer isteyen bir oyun gelirse bu başlıkları `/play/` üzerinde sunucunun göndermesi gerekir. Bugün hiçbir şey bozulmuyor, çünkü o worker'lar zaten kurulamıyor (bkz. Tespit 1).

### CSP

`internal/httpserver/security.go` içindeki `publicCSP`'ye `worker-src 'self'` eklenir. `manifest-src 'self'` 2026-08-14'te zaten eklendi.

## Sayfa Tarafı

### Kontrol ve durumlar

Oyun sayfasında Oyna'nın yanında, `offline.js` tarafından enjekte edilir. JavaScript yoksa hiç görünmez — özellik doğası gereği betik gerektirir ve sitenin konvansiyonu zaten "yalnızca iletişim formu ve konsol betik ister".

| Durum | Görünen |
|---|---|
| yok | "Çevrimdışı oynanabilir yap — 44 MB" |
| iniyor | İlerleme + iptal |
| hazır | "Çevrimdışı hazır — 44 MB" + kaldır |
| bayat | "Yeni sürüm var — 46 MB" + güncelle (eski kopya çalışmaya devam eder) |
| başarısız | Düz bir hata mesajı |

Durum, `caches.keys()` içindeki `game-{slug}-*` girdisiyle build uç noktasının verdiği sürüm karşılaştırılarak bulunur. Çevrimdışıyken uç noktaya ulaşılamaz: cache varsa **hazır** denir, **bayat** denmez — doğrulanamayan şey iddia edilmez.

### Tamamlanma işareti

Dosyalar `game-{slug}-{version}` cache'ine yazılır; ilerleme manifestteki bayt sayılarından okunur (akıştan değil — sayılar zaten kesin).

Bir cache ancak **tamamlanma işareti** varsa geçerli sayılır. İşaret, build'in kendi dosyalarıyla çakışmayacak ayrılmış bir anahtar altında (`/play/{slug}/__offline-complete`) tutulan, gövdesi sürüm damgası olan bir `Response`'tur ve **en son** yazılır. Bu olmadan yarıda kesilmiş bir indirme "hazır" görünür ve oyun uçakta açılmaz. İşareti olmayan cache bir sonraki ziyarette silinir. Worker `/play/*` isteklerini karşılarken bu anahtarı asla dışarı vermez.

Güncelleme aynı kuralla yürür: yeni sürüm tamamlanmadan eski cache silinmez, yani yarım kalan bir güncelleme çalışan kopyayı götürmez. Bu, "eski kopya çalışmaya devam etsin" kararının doğrudan uygulanışıdır.

### Kota

İndirmeden önce `navigator.storage.estimate()` ile yer kontrol edilir; yetmiyorsa yarı yolda patlamak yerine baştan açık bir mesaj verilir. `persist()` bir kez istenir, böylece tarayıcı kopyayı atmaya daha az meyilli olur. İndirme sırasında `QuotaExceededError` gelirse yarım cache silinir ve başarısız durumu gösterilir.

Bir oyunu indirmek **sayfasını ve üzerindeki görselleri de** cache'ler; yoksa "çevrimdışı hazır" derken oyunun kendi sayfası çevrimdışı açılmazdı.

## Test Stratejisi

**Public sitenin JavaScript'inin bugün hiç testi yok** — `arcade.js`, `carousel.js`, `cases.js`, `contact.js`, `lightbox.js`, `osd.js`; vitest yalnızca `admin-ui` tarafında kurulu. Service worker bu repodaki en durumlu bileşen olacak, testsiz göndermek hata olur.

- **JS:** Public site betikleri için repo kökünde küçük bir npm projesi (`package.json` + vitest) kurulur ve `Makefile`'ın `test` hedefine `admin-ui`'nınkinin yanına eklenir. `admin-ui`'nin vitest'ini paket sınırının dışına doğrultmak yerine ayrı proje: ikisinin bağımlılıkları ve tsconfig'i ortak değil. Kökte bugün boş bir `node_modules` duruyor ve `.gitignore`'da **yok** — proje kurulurken ignore edilmeli. Worker'ın karar veren kısımları — rota sınıflandırma, cache adı ayrıştırma, kabuk farkı, indirme durum makinesi — tarayıcı gerektirmeyen saf fonksiyonlara ayrılıp test edilir. `install`/`activate`/`fetch` kancaları ince tutkal kalır ve elle kontrol listesiyle doğrulanır.
- **Go:** extractor manifesti ve çöp dışlama; sürümün aynı arşiv için değişmediği; migration 0021; build uç noktası (yayınlanmamış, build'siz, var olmayan slug); shell uç noktası; `/sw.js` rotası ve başlıkları; `publicCSP`'de `worker-src`.

## Kapsam Dışı

Background sync, push bildirimleri, iletişim formunun çevrimdışı kuyruğa alınması, genel bir depolama yönetim ekranı, ve indirilmemiş oyunların sayfalarının önden cache'lenmesi.

## Bilinen Riskler

- **iOS agresif tahliye eder** ve `persist()` orada verilmez; 90 MB'lık bir oyun haber vermeden kaybolabilir. Arayüz bunu vaat etmemeli.
- **Sekiz oyunun tamamı 386 MB eder;** bazı cihazlarda tarayıcı çok daha öncesinde reddeder. Kota kontrolü bunu nazikçe karşılar, ama sınırın kendisi kalkmaz.
- **Kapsam geri alınabilir:** ileride doğru export edilmiş bir Godot build'i `/play/{slug}/`'i geri isteyebilir. Extraction worker dosyasını sildiği için bugün korunuyoruz; motor bir gün worker'ı satır içine gömerse bu korunma düşer.

## Sonraki Adım

Bu spec'ten bir implementasyon planı yazılacak (writing-plans). Sıra önerisi: build manifesti + migration + uç noktalar (Go, tek başına test edilebilir) → worker ve kabuk uç noktası → indirme arayüzü → vitest kurulumu.
