# 🌌 Pixerve

**Pixerve** is a work-in-progress, open-source image ingestion and processing server built for modern web services.  
It aims to make image handling **predictable, transparent, and fast** — without locking anyone into closed APIs or external services.

---

## 🧭 Motivation

Every growing product ends up building the same thing over and over again:  
a simple, reliable place to upload images, convert them, resize them, and serve them.  

Pixerve is our attempt to stop rewriting that wheel — and do it properly this time:  
a small, efficient image worker you can host, shape, and extend however you like.

---

## 🌱 Goals

- 🧩 **Simple but powerful** — upload, queue, process, and serve with minimal setup.  
- 🔒 **Secure by design** — credentials never travel in JWTs; Pixerve only stores reference keys.  
- ⚙️ **Extensible** — supports multiple storage backends and formats; built for composability.  
- 🌍 **Self-hosted** — runs wherever Go does; no external dependencies beyond standard tools.  
- 🧠 **Transparent** — open specs, clear job structures, human-readable configuration.

---

## 🚧 Work in Progress

Pixerve is under active design and development.  
There are no releases yet — only rough code and ideas gradually turning into something useful.

---

## 🗓️ Milestones

| Phase | Description | Status |
|-------|--------------|--------|
| **0.1 – Core Foundation** | Basic job structure, JWT schema, and local image processing. | 🧩 In progress |
| **0.2 – Storage Abstraction** | Add S3 and SFTP backends with credential registry via Pebble. | ⚙️ Planned |
| **0.3 – Worker & Queue System** | Reliable job orchestration, rate limiting, and callback logic. | 🛠️ Planned |
| **0.4 – Direct Host Module** | Serve processed assets directly with tenant subdirectory logic. | 🧱 Planned |
| **0.5 – Public Preview** | Minimal viable open-source release for early feedback. | 🌱 Tentative |

---

## 🧭 Philosophy

Pixerve doesn’t chase feature lists or buzzwords.  
Its purpose is **clarity and control** — the kind of tool you can actually understand, trust, and build around.  

If it ends up helping others the same way it helps us, then it’s already a success.

---

## 🪶 License

TBD — open source, permissive, and community-friendly once the first release lands.

---

## 🗣️ Follow Along

Pixerve’s journey is open.  
Discussions, design notes, and ideas will all live here — transparently, as they evolve.  

Stay tuned, and if you like where this is going, star the repo or drop by with ideas.  

> _“ Managed Image Processing — one pixel at a time.”_
