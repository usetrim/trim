-- Fail closed when .trimrc custom_query needs Tree-sitter but binary is CGO-free.
-- Receipt first-party PDF download chrome (filename fmt + failure toast).

insert into public.site_messages (code, body) values
  ('CLI_TREESITTER_REQUIRED', 'custom_query in .trimrc requires a Tree-sitter Trim build (CGO_ENABLED=1 go build -tags treesitter). Default CGO-free binaries cannot apply Mode 4 queries.'),
  ('RECEIPT_DOWNLOAD_PDF_FAILED', 'Could not download the receipt PDF. Try again or use Print.'),
  ('RECEIPT_PDF_FILENAME_FMT', 'receipt-%s.pdf')
on conflict (code) do nothing;
