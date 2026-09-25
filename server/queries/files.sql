-- name: ConversationFileNames :many
-- The names already used in one of a conversation's two folders, so a new
-- file can be given one that is free.
SELECT name FROM conversation_files
WHERE conversation_id = @conversation_id AND origin = @origin;

-- name: CreateConversationFile :one
INSERT INTO conversation_files (conversation_id, entry_id, origin, name, media_type, size, sha256)
VALUES (@conversation_id, @entry_id, @origin, @name, @media_type, @size, @sha256)
RETURNING *;

-- name: PutConversationFileContent :exec
INSERT INTO conversation_file_contents (file_id, content) VALUES (@file_id, @content);

-- name: ListConversationFiles :many
-- Newest first, which is the order the pane shows them in.
SELECT * FROM conversation_files
WHERE conversation_id = @conversation_id
ORDER BY created_at DESC, name;

-- name: ConversationFileFor :one
-- One file, only if it is in a conversation this account can see. The
-- ownership check for every read of a file, from the browser or a computer.
SELECT f.* FROM conversation_files f
JOIN conversations c ON c.id = f.conversation_id
WHERE f.id = @id
  AND EXISTS (
      SELECT 1 FROM workspace_members m
      WHERE m.workspace_id = c.workspace_id AND m.user_id = @user_id
  );

-- name: ConversationFileContent :one
SELECT content FROM conversation_file_contents WHERE file_id = @file_id;

-- name: AttachConversationFiles :many
-- Sending a message takes the uploads waiting in its box. Only this
-- conversation's, only uploads, and only ones not already sent with another.
UPDATE conversation_files
SET entry_id = @entry_id
WHERE id = ANY(@ids::uuid[])
  AND conversation_id = @conversation_id
  AND origin = 'upload'
  AND entry_id IS NULL
RETURNING *;

-- name: DeleteConversationFileContent :exec
DELETE FROM conversation_file_contents WHERE file_id = @file_id;

-- name: DeleteUnsentConversationFile :execrows
-- Removing an upload from the box. Once sent, a file is part of the record.
DELETE FROM conversation_files WHERE id = @id AND entry_id IS NULL AND origin = 'upload';

-- name: DeleteConversationFileContents :exec
DELETE FROM conversation_file_contents
WHERE file_id IN (SELECT id FROM conversation_files WHERE conversation_id = @conversation_id);

-- name: DeleteConversationFiles :exec
DELETE FROM conversation_files WHERE conversation_id = @conversation_id;

-- name: DeleteEveryConversationFileContent :exec
DELETE FROM conversation_file_contents;

-- name: DeleteEveryConversationFile :exec
DELETE FROM conversation_files;
