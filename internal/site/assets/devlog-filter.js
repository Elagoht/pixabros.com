// The devlog sidebar filter. The page is statically rendered, so filtering is
// purely presentational: rows are hidden, never re-fetched. Without this
// script the sidebar buttons do nothing and every post stays listed.
(() => {
  const root = document.querySelector("[data-devlog-filters]");
  const rows = Array.from(document.querySelectorAll("[data-game], [data-year]"));
  const empty = document.querySelector("[data-filter-empty]");
  if (!root || rows.length === 0) return;

  let game = "";
  let year = "";

  const matches = (row) =>
    (!game || row.dataset.game === game) && (!year || row.dataset.year === year);

  const apply = () => {
    let shown = 0;
    for (const row of rows) {
      const hit = matches(row);
      row.hidden = !hit;
      if (hit) shown++;
    }
    if (empty) empty.hidden = shown > 0;
  };

  const setActive = (clicked) => {
    for (const button of root.querySelectorAll("[data-filter-game]")) {
      button.classList.toggle("is-active", button === clicked);
    }
  };

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
