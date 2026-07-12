CREATE TYPE user_role AS ENUM ('guest', 'user', 'moderator', 'admin');
CREATE TYPE university AS ENUM ('sechenov', 'pirogov', 'evdokimov', 'other');
CREATE TYPE textbook_storage_type AS ENUM ('A', 'B');
CREATE TYPE textbook_license_type AS ENUM ('cc_by', 'cc_by_nc', 'cc_by_sa', 'cc0', 'public_domain', 'all_rights_reserved', 'custom');
CREATE TYPE card_task_status AS ENUM ('pending', 'processing', 'done', 'failed');
CREATE TYPE card_task_source_type AS ENUM ('catalog_textbook', 'user_upload');
CREATE TYPE card_difficulty AS ENUM ('easy', 'medium', 'hard');
CREATE TYPE poi_type AS ENUM ('coworking', 'cafe', 'library', 'canteen', 'park', 'other');
CREATE TYPE thread_tag AS ENUM ('study', 'department', 'humor', 'marketplace', 'clinical_base', 'news', 'help', 'other');
CREATE TYPE report_status AS ENUM ('pending', 'reviewed', 'dismissed');
CREATE TYPE subscription_target_type AS ENUM ('thread', 'user', 'tag');
CREATE TYPE reaction_target_type AS ENUM ('thread', 'comment');
CREATE TYPE audit_action AS ENUM (
    'user_ban', 'user_unban', 'user_role_change',
    'thread_hide', 'thread_unhide', 'thread_delete',
    'comment_hide', 'comment_unhide', 'comment_delete',
    'poi_create', 'poi_update', 'poi_delete',
    'textbook_create', 'textbook_update', 'textbook_delete'
    );
CREATE TYPE notification_type AS ENUM (
    'thread_reply', 'comment_reply', 'reaction',
    'card_task_done', 'card_task_failed',
    'moderation_action', 'system'
    );
