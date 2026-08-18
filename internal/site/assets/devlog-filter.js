// The devlog index filter. Rows are served by the server: the first page is
// in the HTML, and every further page -- and every search, game and year
// narrowing -- comes from /api/devlog/posts, debounced and one page at a time.
// Without this script the sidebar buttons do nothing, the load-more button is
// hidden, and the statically rendered first page is the whole devlog.
(() => {
  const feed = document.querySelector("[data-devlog-feed]");
  if (!feed) return;
  const root = document.querySelector("[data-devlog-filters]");
  const queryInput = document.querySelector("[data-devlog-query]");
  const empty = feed.querySelector("[data-filter-empty]");
  const more = feed.querySelector("[data-devlog-more]");

  let list = feed.querySelector(".post-list");
  let featured = feed.querySelector(".post-feature");
  const idleEmpty = empty ? empty.textContent : "";

  let page = 1;
  let perPage = Number(feed.dataset.perPage) || 10;
  let total = Number(feed.dataset.total) || 0;
  let q = "";
  let game = "";
  let year = "";
  let busy = false;
  let controller = null;
  let timer = null;

  const MONTHS = ["January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December"];

  // formatDate turns a YYYY-MM-DD into the same "2 January 2006" the server
  // renders, so a fetched row reads exactly like one that was in the page.
  const formatDate = (value) => {
    const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value || "");
    if (!match) return value;
    return `${Number(match[3])} ${MONTHS[Number(match[2]) - 1]} ${match[1]}`;
  };

  const rowFor = (post) => {
    const link = document.createElement("a");
    link.className = "post-row";
    link.href = "/devlog/" + post.slug;

    if (post.image) {
      const img = document.createElement("img");
      img.className = "post-row__thumb";
      img.src = post.image;
      img.alt = "";
      img.loading = "lazy";
      link.appendChild(img);
    } else {
      const placeholder = document.createElement("span");
      placeholder.className = "post-row__thumb post-row__thumb--empty";
      placeholder.setAttribute("aria-hidden", "true");
      link.appendChild(placeholder);
    }

    const text = document.createElement("span");
    text.className = "post-row__text";

    const meta = document.createElement("span");
    meta.className = "post-row__meta-line";
    if (post.date) {
      const date = document.createElement("span");
      date.textContent = formatDate(post.date);
      meta.appendChild(date);
    }
    if (post.game) {
      const game = document.createElement("span");
      game.className = "post-row__game";
      game.textContent = post.game;
      meta.appendChild(game);
    }
    text.appendChild(meta);

    const title = document.createElement("span");
    title.className = "post-row__title";
    title.textContent = post.title;
    text.appendChild(title);

    link.appendChild(text);

    const item = document.createElement("li");
    item.appendChild(link);
    return item;
  };

  const hasMore = () => page * perPage < total;

  const updateMore = () => {
    if (more) more.hidden = !hasMore();
  };

  // request fetches one page of the search API, aborting any request it
  // supersedes so a fast typist never gets an out-of-order list.
  const request = async (pageNum) => {
    if (controller) controller.abort();
    controller = new AbortController();
    const params = new URLSearchParams();
    if (q) params.set("q", q);
    if (game) params.set("game", game);
    if (year) params.set("year", year);
    params.set("page", String(pageNum));
    params.set("per_page", String(perPage));
    try {
      const res = await fetch("/api/devlog/posts?" + params, {
        signal: controller.signal,
        headers: { Accept: "application/json" },
      });
      if (!res.ok) return null;
      return await res.json();
    } catch (err) {
      if (err && err.name === "AbortError") return null;
      return null;
    }
  };

  // run fetches a page. fresh replaces the list (a new search or filter);
  // otherwise the page is appended, which is how load-more works.
  const run = async (fresh) => {
    if (busy) return;
    busy = true;
    const data = await request(fresh ? 1 : page + 1);
    busy = false;
    if (!data) return;

    page = data.page;
    perPage = data.per_page || perPage;
    total = data.total;

    if (fresh) {
      if (featured) featured.remove();
      if (list) list.remove();
      list = document.createElement("ul");
      list.className = "post-list";
      feed.insertBefore(list, empty);
      for (const post of data.posts) list.appendChild(rowFor(post));
      if (empty) {
        empty.textContent = data.posts.length === 0 ? (q ? "No logs match the query." : idleEmpty) : idleEmpty;
        empty.hidden = data.posts.length !== 0;
      }
    } else {
      if (!list) return;
      for (const post of data.posts) list.appendChild(rowFor(post));
    }
    updateMore();
  };

  const setActive = (clicked) => {
    for (const button of root.querySelectorAll("[data-filter-game]")) {
      button.classList.toggle("is-active", button === clicked);
    }
  };

  if (queryInput) {
    queryInput.addEventListener("input", () => {
      clearTimeout(timer);
      timer = setTimeout(() => {
        q = queryInput.value.trim().toLowerCase();
        run(true);
      }, 250);
    });
  }

  for (const button of root.querySelectorAll("[data-filter-game]")) {
    button.addEventListener("click", () => {
      game = button.dataset.filterGame || "";
      setActive(button);
      run(true);
    });
  }
  for (const button of root.querySelectorAll("[data-filter-year]")) {
    button.addEventListener("click", () => {
      year = year === button.dataset.filterYear ? "" : button.dataset.filterYear;
      button.classList.toggle("is-active", year !== "");
      run(true);
    });
  }

  if (more) {
    more.addEventListener("click", () => run(false));
  }

  // On the first page the server already drew, only the load-more state needs
  // deciding: is there a second page at all?
  updateMore();
})();