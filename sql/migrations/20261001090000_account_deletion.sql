-- migrate:up
-- Deleting an account anonymises it rather than removing the row: rentals and
-- rental_checkout reference the customer, and fields, plots and their rentals
-- reference the farm, and those are payment records that must outlive the
-- person (and other customers' rental history hangs off a farmer's plots). Set
-- means the account's personal data has been scrubbed and it can no longer sign
-- in or be found; see AccountRepository.DeleteAccount.
ALTER TABLE account ADD COLUMN deleted_at TIMESTAMPTZ;

-- migrate:down
ALTER TABLE account DROP COLUMN deleted_at;
