# Final whole-branch review — issue #8 (2026-08-29 01:4x+08:00)

Branch codex/admin-config-08, 5a4666ec7..fceb8392e, 4 commits
(812ed28ed docs / 70bc7e530 T1 schema / 8b71689c4 T2 editor / fceb8392e T3 i18n).
Full diff re-read end-to-end against plan + prescription + AGENTS.md frontend
conventions.

- Commit sequence builds independently (each task commit verified by
  build+lint gate before next; trio re-run at HEAD).
- No secrets, no debug residue, no console output added.
- Schema: tier rules exhaustive; error paths precise (row-level); i18n-key
  messages with FieldError translation — repo-est. pattern preserved.
- Editor: hooks rules respected (TierEditor is a named component, no hooks in
  map; useFieldArray/useWatch only). resetPrice tiered default = graduated +
  1 tier from 0 — safe default matching first-open rules.
- Mapping: firstUnit intentionally excluded from wire (up_to_amount implicit
  ranges) — matches spec & prescription note; per-tier unitPrice always,
  flatPrice omitted when empty (SDK doc: ≥1 component required).
- Risks accepted (Minor, non-blocking): (1) switching price kind discards
  entered tiers (prescribed reset semantics, same as #7 amount reset);
  (2) volume/graduated mode copy is long in the select dropdown (prescription
  text verbatim); (3) no reverse mapping (draft→tiered backfill) — #9 scope.
- Reviews: spec PASS (0 fix rounds), quality PASS. Walkthrough evidence
  /tmp/issue8-shots (3 png, non-blank 1280×720).

Verdict: PASS — track LOCALLY COMPLETE; fix rounds used 0/5.
