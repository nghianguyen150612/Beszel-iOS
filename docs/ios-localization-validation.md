# P015 localization validation

## Catalog build and registered locales

The final clean build ran Lingui extraction and compilation for all 32 registered locales. Vite emitted locale modules/chunks, and the frontend asset validator confirmed they resolve and have generated assets.

Locales: en, ar, bg, cs, da, de, el, es, fa, fr, he, hr, hu, id, it, ja, ko, nl, no, pl, pt, tr, ru, sl, sr, sv, uk, uz, vi, zh, zh-CN, zh-HK.

Lingui reported 537 source messages. Missing translations:

| Locale | Missing | Locale | Missing |
| --- | ---: | --- | ---: |
| en | 0 (source) | ar | 3 |
| bg | 5 | cs | 19 |
| da | 35 | de | 35 |
| el | 8 | es | 20 |
| fa | 3 | fr | 37 |
| he | 5 | hr | 18 |
| hu | 13 | id | 28 |
| it | 24 | ja | 5 |
| ko | 6 | nl | 36 |
| no | 30 | pl | 20 |
| pt | 25 | ru | 5 |
| sl | 13 | sr | 5 |
| sv | 27 | tr | 5 |
| uk | 4 | uz | 22 |
| vi | 8 | zh | 8 |
| zh-CN | 5 | zh-HK | 4 |

Vietnamese has 529/537 messages translated. Browser checks rendered Vietnamese Unicode text, selected it in settings, reloaded, and confirmed persistence. Browser locale vi-VN selected Vietnamese. An unsupported stored locale fell back to English and replaced the invalid value.

## RTL checks

Arabic, Persian, and Hebrew switching set the document language and direction. Browser computed style confirmed RTL navigation, table, alert, and dialog. Seven system charts were present on the RTL system-detail page. Switching back to English restored English document language.

## Translator preservation

Lingui extraction generated updated message references and missing entries during the build. Generated PO changes were inspected and restored after builds, following repository CI practice. No translator text was deliberately edited or removed. Incomplete locale counts remain visible; this is not complete translation coverage.
