CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_name TEXT NOT NULL,
    owner_id INTEGER NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES users(id),
    UNIQUE(repo_name, owner_id)
);

CREATE TABLE IF NOT EXISTS issues (
    id INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL,
    number INTEGER,
    creater_id INTEGER,
    processor_id INTEGER,
    create_time TIMESTAMP,
    close_time TIMESTAMP,
    status TEXT CHECK(status IN ('close', 'open')),
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    FOREIGN KEY (creater_id) REFERENCES users(id),
    FOREIGN KEY (processor_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_issues_repo_id ON issues(repo_id);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_create_time ON issues(create_time);
CREATE INDEX IF NOT EXISTS idx_issues_close_time ON issues(close_time);
CREATE INDEX IF NOT EXISTS idx_issues_creater_id ON issues(creater_id);

CREATE TABLE IF NOT EXISTS pulls (
    id INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL,
    number INTEGER NOT NULL,
    creater_id INTEGER,
    status TEXT CHECK(status IN ('merged', 'closed', 'open')),
    processor_id INTEGER,
    create_time TIMESTAMP,
    merged_at TIMESTAMP,
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    FOREIGN KEY (creater_id) REFERENCES users(id),
    FOREIGN KEY (processor_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_pulls_merged_at ON pulls(merged_at);
CREATE INDEX IF NOT EXISTS idx_pulls_repo_id ON pulls(repo_id);
CREATE INDEX IF NOT EXISTS idx_pulls_status ON pulls(status);
CREATE INDEX IF NOT EXISTS idx_pulls_create_time ON pulls(create_time);
CREATE INDEX IF NOT EXISTS idx_pulls_creater_id ON pulls(creater_id);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    UNIQUE(repo_id, name)
);

CREATE INDEX IF NOT EXISTS idx_tags_repo_id ON tags(repo_id);

CREATE TABLE IF NOT EXISTS issue_tags (
    issue_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_issue_tags_issue_id ON issue_tags(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_tags_tag_id ON issue_tags(tag_id);

CREATE TABLE IF NOT EXISTS pull_tags (
    pull_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (pull_id) REFERENCES pulls(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (pull_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_pull_tags_pull_id ON pull_tags(pull_id);
CREATE INDEX IF NOT EXISTS idx_pull_tags_tag_id ON pull_tags(tag_id);
