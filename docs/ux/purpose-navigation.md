# Purpose navigation and complete Work discovery

An Outcome has six primary areas: Overview, Plan, Work, Issues, Evidence and Activity. Collections live beneath their purpose; operational Task projections are status filters inside Work. List and Board are Work views. Workspace settings remain separate and require administration permission.

Overview explains persisted Draft preparation and offers explicit actions to define criteria, add Objectives/Tasks and create a Roadmap. Activating an Outcome is a separate command whose prerequisites remain server-enforced. A deterministic suggested next step explains the persisted condition behind its suggestion and executes nothing automatically.

Deep links encode authorized workspace, Outcome, area, view, section and optional typed item. They contain no credential and grant no access. Reload restores them after authentication; browser Back/Forward restores area and detail. Copy link produces a canonical context URL. The native skip link retains its fragment without interfering with query routing.

The All Tasks list searches every authorized Task title and priority through the bounded reference query, then hydrates only the matching page. Metadata and entity reads must agree on Outcome revision. The query, scope, priority and Outcome revision bind its cursor; changing one requires a new search. The Board and operational status filters use coherent projections and clearly identify loaded-page filtering. Other collection filters also label their loaded-page scope. No loaded data is presented as an exhaustive search.

New `priority` filtering is additive and valid only for Task metadata queries. Memory, SQLite and real PostgreSQL tests cover an off-page high-priority Task among 1,001 records, invalid kinds and cursor/filter mismatch. Effective permission hints also have an actual SDK HTTP/MCP parity check; these remain presentation metadata, never authority.

Automated acceptance covers six areas, Draft persistence, complete Work search, deep-link reload/back/forward and protocol regressions. Representative-user usability and manual screen-reader validation remain pending by owner decision.
