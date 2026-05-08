-- Minimal Mattermost-shaped schema covering the tables that the Insights
-- store queries read. This is a hand-crafted subset — only the columns and
-- tables our queries reference are present. As we add more insights, extend
-- this schema with the columns they need.
--
-- All identifiers use lowercase, since unquoted CamelCase identifiers in the
-- ported queries are normalized to lowercase by PostgreSQL. The columns and
-- types here mirror the production Mattermost schema's behavior for the
-- subset we exercise.

CREATE TABLE channels (
    id           varchar(26) PRIMARY KEY,
    type         char(1)     NOT NULL,
    teamid       varchar(26) NOT NULL DEFAULT '',
    displayname  varchar(64) NOT NULL DEFAULT '',
    name         varchar(64) NOT NULL DEFAULT '',
    createat     bigint      NOT NULL DEFAULT 0,
    deleteat     bigint      NOT NULL DEFAULT 0
);

CREATE TABLE channelmembers (
    channelid    varchar(26) NOT NULL,
    userid       varchar(26) NOT NULL,
    PRIMARY KEY (channelid, userid)
);

CREATE TABLE publicchannels (
    id           varchar(26) PRIMARY KEY,
    teamid       varchar(26) NOT NULL,
    displayname  varchar(64) NOT NULL DEFAULT '',
    name         varchar(64) NOT NULL DEFAULT '',
    deleteat     bigint      NOT NULL DEFAULT 0
);

CREATE TABLE reactions (
    postid       varchar(26) NOT NULL,
    userid       varchar(26) NOT NULL,
    emojiname    varchar(64) NOT NULL,
    channelid    varchar(26) NOT NULL DEFAULT '',
    createat     bigint      NOT NULL,
    deleteat     bigint      NOT NULL DEFAULT 0,
    PRIMARY KEY (postid, userid, emojiname)
);

CREATE TABLE posts (
    id           varchar(26) PRIMARY KEY,
    userid       varchar(26) NOT NULL,
    channelid    varchar(26) NOT NULL,
    rootid       varchar(26) NOT NULL DEFAULT '',
    createat     bigint      NOT NULL DEFAULT 0,
    updateat     bigint      NOT NULL DEFAULT 0,
    deleteat     bigint      NOT NULL DEFAULT 0,
    type         varchar(26) NOT NULL DEFAULT '',
    props        jsonb
);

CREATE TABLE threads (
    postid          varchar(26) PRIMARY KEY,
    channelid       varchar(26) NOT NULL,
    replycount      bigint      NOT NULL DEFAULT 0,
    lastreplyat     bigint      NOT NULL DEFAULT 0,
    participants    jsonb,
    threaddeleteat  bigint
);

CREATE TABLE threadmemberships (
    postid       varchar(26) NOT NULL,
    userid       varchar(26) NOT NULL,
    following    boolean     NOT NULL DEFAULT FALSE,
    PRIMARY KEY (postid, userid)
);

CREATE TABLE users (
    id                  varchar(26) PRIMARY KEY,
    username            varchar(64) NOT NULL DEFAULT '',
    firstname           varchar(64) NOT NULL DEFAULT '',
    lastname            varchar(64) NOT NULL DEFAULT '',
    nickname            varchar(64) NOT NULL DEFAULT '',
    position            varchar(128) NOT NULL DEFAULT '',
    lastpictureupdate   bigint      NOT NULL DEFAULT 0,
    deleteat            bigint      NOT NULL DEFAULT 0
);

CREATE TABLE teammembers (
    teamid       varchar(26) NOT NULL,
    userid       varchar(26) NOT NULL,
    createat     bigint      NOT NULL DEFAULT 0,
    deleteat     bigint      NOT NULL DEFAULT 0,
    PRIMARY KEY (teamid, userid)
);

CREATE TABLE bots (
    userid       varchar(26) PRIMARY KEY,
    deleteat     bigint      NOT NULL DEFAULT 0
);

-- focalboard tables (subset needed for Top Boards). Faithfully shaped to
-- the production focalboard schema (mattermost-plugin-boards migrations
-- 000001, 000018) so the deprecated SQL ports without modification.
-- Tables are prefixed `focalboard_` exactly as the Boards plugin
-- configures (`DBTablePrefix = "focalboard_"`).

CREATE TABLE focalboard_boards (
    id            varchar(36) NOT NULL PRIMARY KEY,
    insert_at     timestamptz NOT NULL DEFAULT NOW(),
    team_id       varchar(36) NOT NULL,
    channel_id    varchar(36),
    created_by    varchar(36),
    modified_by   varchar(36),
    type          varchar(1)  NOT NULL,
    title         text        NOT NULL,
    icon          varchar(256),
    is_template   boolean,
    create_at     bigint,
    update_at     bigint,
    delete_at     bigint
);

CREATE TABLE focalboard_boards_history (
    id            varchar(36) NOT NULL,
    insert_at     timestamptz NOT NULL DEFAULT NOW(),
    team_id       varchar(36) NOT NULL,
    channel_id    varchar(36),
    created_by    varchar(36),
    modified_by   varchar(36),
    type          varchar(1)  NOT NULL,
    title         text        NOT NULL,
    icon          varchar(256),
    is_template   boolean,
    create_at     bigint,
    update_at     bigint,
    delete_at     bigint,
    PRIMARY KEY (id, insert_at)
);

CREATE TABLE focalboard_blocks_history (
    id            varchar(36) NOT NULL,
    insert_at     timestamptz NOT NULL DEFAULT NOW(),
    board_id      varchar(36),
    modified_by   varchar(36),
    type          text,
    create_at     bigint,
    update_at     bigint,
    delete_at     bigint,
    PRIMARY KEY (id, insert_at)
);

CREATE TABLE focalboard_board_members (
    board_id      varchar(36) NOT NULL,
    user_id       varchar(36) NOT NULL,
    scheme_admin  boolean,
    PRIMARY KEY (board_id, user_id)
);

-- mattermost-plugin-playbooks tables (subset needed for Top Playbooks).
-- Faithful to the production schema (`server/sqlstore/migrations/postgres/`
-- 000002, 000003, 000004 plus the Go-coded migration that adds the
-- `Public` column on IR_Playbook). Top Playbooks queries reference
-- IR_Playbook (p), IR_Incident (i), and IR_PlaybookMember (pm).

CREATE TABLE IR_Playbook (
    ID                     text   PRIMARY KEY,
    Title                  text   NOT NULL,
    Description            text   NOT NULL DEFAULT '',
    TeamID                 text   NOT NULL,
    CreatePublicIncident   boolean NOT NULL DEFAULT FALSE,
    CreateAt               bigint NOT NULL,
    DeleteAt               bigint NOT NULL DEFAULT 0,
    Public                 boolean DEFAULT FALSE
);

CREATE TABLE IR_PlaybookMember (
    PlaybookID  text NOT NULL REFERENCES IR_Playbook(ID),
    MemberID    text NOT NULL,
    UNIQUE (PlaybookID, MemberID)
);

CREATE TABLE IR_Incident (
    ID         text   PRIMARY KEY,
    PlaybookID text   NOT NULL DEFAULT '',
    TeamID     text   NOT NULL,
    CreateAt   bigint NOT NULL,
    DeleteAt   bigint NOT NULL DEFAULT 0
);
