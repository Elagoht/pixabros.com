// The devlog sidebar filter. The page is statically rendered, so filtering is
// purely presentational: rows are hidden, never re-fetched. Without this
// script the sidebar buttons do nothing and every post stays listed.
(() => {
  const root = document.querySelector("[data-devlog-filters]");
  const rows = Array.from(document.querySelectorAll("[data-game], [data-year]"));
  const empty = document.querySelector("[data-filter-empty]");
  const queryInput = document.querySelector("[data-devlog-query]");
  if (!root || rows.length === 0) return;

  let game = "";
  let year = "";
  let query = "";
  const idleEmpty = empty ? empty.textContent : "";

  const titleOf = (row) => {
    const title = row.querySelector(".post-row__title, .post-feature__title");
    return title ? title.textContent : row.textContent;
  };

  const matches = (row) =>
    (!query || titleOf(row).toLowerCase().includes(query)) &&
    (!game || row.dataset.game === game) &&
    (!year || row.dataset.year === year);

  const apply = () => {
    let shown = 0;
    for (const row of rows) {
      const hit = matches(row);
      row.hidden = !hit;
      if (hit) shown++;
    }
    if (empty) {
      empty.hidden = shown > 0;
      empty.textContent = query && shown === 0 ? "No logs match the query." : idleEmpty;
    }
  };

  const setActive = (clicked) => {
    for (const button of root.querySelectorAll("[data-filter-game]")) {
      button.classList.toggle("is-active", button === clicked);
    }
  };

  if (queryInput) {
    queryInput.addEventListener("input", () => {
      query = queryInput.value.trim().toLowerCase();
      apply();
    });
  }

  for (const button of root.querySelectorAll("[data-filter-game]")) {
    button.addEventListener("click", () => {
      game = button.dataset.filterGame || "";
      setActive(button);
      apply();
    });
  }
  for (const button of root.querySelectorAll("[data-filter-year]")) {
    button.addEventListener("click", () => {
      year = year === button.dataset.filterYear ? "" : button.dataset.filterYear;
      button.classList.toggle("is-active", year !== "");
      apply();
    });
  }
})();
