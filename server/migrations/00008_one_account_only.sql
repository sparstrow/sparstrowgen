-- +goose Up
-- The database, not the application, decides that there is exactly one account.
--
-- Sign-up checked "does a user already exist?" and then inserted, with a
-- comment claiming the UNIQUE constraint on email made that safe. It does not.
-- That constraint only arbitrates between two people claiming the SAME address;
-- two simultaneous sign-ups with different emails both pass the count check,
-- both insert, and the deployment permanently has two owners — each able to run
-- agents on the machine.
--
-- A unique index on a constant expression is the whole fix: every row produces
-- the same key, so the second INSERT cannot land whatever it contains. The race
-- is then decided where races have to be decided, by the database, rather than
-- by a check the application performs a few milliseconds earlier.
--
-- This is what stands between a stranger who obtained a setup code and a
-- permanent second account, so it is worth it being a constraint rather than a
-- convention.
CREATE UNIQUE INDEX users_only_one ON users ((true));

-- +goose Down
DROP INDEX users_only_one;
