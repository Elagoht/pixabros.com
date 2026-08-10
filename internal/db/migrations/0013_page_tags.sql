CREATE TABLE page_tags (
    page_key TEXT NOT NULL REFERENCES rendered_pages(page_key) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (page_key, tag)
);

CREATE INDEX idx_page_tags_tag ON page_tags(tag);
