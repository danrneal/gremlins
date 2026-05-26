---
title: Invert boolean literals
---

This mutation type inverts boolean literal values `true` and `false`.

| From    | To      |
|:-------:|:-------:|
| `true`  | `false` |
| `false` | `true`  |

This mutator is particularly useful for catching untested state
assignments (e.g., `isReady = true`) and exposing missing negative test
paths when functions unconditionally return a boolean literal.
