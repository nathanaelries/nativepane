# Layout claim coverage

This index covers README first-release layout and ROADMAP rendering work.
Approximate means the existing NativePane view; planned means a limitation is
captured, not implemented. Preserved means source package data survives saves.
The manifest gate requires actual OOXML evidence for every assigned claim.
Semantic snapshots and strict browser pixels protect the reviewed current state.
No Office-equivalent layout or new editing feature is asserted.

| Claim | Status | Office fixture | Source construct |
|---|---|---|---|
| Fonts, sizes, bold, italic, color and underline. | approximate | [word-fonts](office/word-fonts.docx) | `word/document.xml: .//w:rFonts`; `word/document.xml: .//w:sz`; `word/document.xml: .//w:b`; `word/document.xml: .//w:i`; `word/document.xml: .//w:color`; `word/document.xml: .//w:u` |
| Document defaults, paragraph/character style inheritance and direct overrides. | approximate | [word-style-inheritance](office/word-style-inheritance.docx) | `word/styles.xml: .//w:docDefaults`; `word/styles.xml: .//w:basedOn`; `word/document.xml: .//w:pStyle`; `word/document.xml: .//w:rStyle` |
| Applied character styles on mixed text runs. | approximate | [word-character-style](office/word-character-style.docx) | `word/document.xml: .//w:rStyle` |
| Theme font/color hints and local font fallback. | approximate | [word-theme](office/word-theme.docx) | `word/styles.xml: .//w:rFonts[@w:asciiTheme]`; `word/theme/*.xml: .//a:fontScheme`; `word/theme/*.xml: .//a:clrScheme` |
| Paragraph alignment, spacing and borders. | approximate | [word-paragraph-format](office/word-paragraph-format.docx) | `word/document.xml: .//w:jc`; `word/document.xml: .//w:spacing`; `word/document.xml: .//w:pBdr` |
| Page dimensions, including landscape geometry. | approximate | [word-landscape](office/word-landscape.docx) | `word/document.xml: .//w:pgSz[@w:orient="landscape"]` |
| Source margins determine printable content width. | approximate | [word-narrow-margins](office/word-narrow-margins.docx) | `word/document.xml: .//w:pgMar` |
| Nested flow tables retain cell/paragraph order. | approximate | [word-nested-tables](office/word-nested-tables.docx) | `word/document.xml: .//w:tc/w:tbl` |
| Horizontal spans and vertical merge groups stay intact. | approximate | [word-merged-table](office/word-merged-table.docx) | `word/document.xml: .//w:gridSpan`; `word/document.xml: .//w:vMerge` |
| Multiple paragraphs per cell are not flattened or duplicated. | approximate | [word-cell-paragraphs](office/word-cell-paragraphs.docx) | `word/document.xml: .//w:tc/w:p (at least 2)` |
| Grid widths and preferred cell widths refine column proportions. | approximate | [word-merged-table](office/word-merged-table.docx) | `word/document.xml: .//w:tblGrid/w:gridCol`; `word/document.xml: .//w:tcW[@w:type="dxa"]` |
| Table shading and cell padding are projected. | approximate | [word-table](office/word-table.docx) | `word/styles.xml: .//w:shd`; `word/styles.xml: .//w:tblCellMar` |
| Direct cell borders are projected. | approximate | [word-conditional-floating](office/word-conditional-floating.docx) | `word/document.xml: .//w:tcBorders` |
| Tables continue between row groups; designated headers repeat. | approximate | [word-long-tables](office/word-long-tables.docx) | `word/document.xml: .//w:tblHeader`; `word/document.xml: .//w:tr (at least 3)` |
| Existing text edits reflow without missing or duplicated fields. | approximate | [word-long-tables](office/word-long-tables.docx) | `word/document.xml: .//w:tblHeader`; `word/document.xml: .//w:t` |
| Oversized paragraphs/row groups expand instead of clipping. | approximate | [word-long-tables](office/word-long-tables.docx) | `word/document.xml: .//w:tr (at least 3)` |
| Keep-with-next is projected. | approximate | [word-keep-next](office/word-keep-next.docx) | `word/document.xml: .//w:keepNext`; `word/document.xml: .//w:pStyle` |
| Applied page-break-before styles start a new page. | approximate | [word-page-break](office/word-page-break.docx) | `word/styles.xml: .//w:style[@w:styleId="Heading1"]/w:pPr/w:pageBreakBefore`; `word/document.xml: .//w:pStyle[@w:val="Heading1"]` |
| Mixed sections retain explicit approximate-layout markers. | planned | [word-mixed-sections](office/word-mixed-sections.docx) | `word/document.xml: .//w:sectPr (at least 2)` |
| List content is retained with numbering-omission markers. | planned | [word-lists](office/word-lists.docx) | `word/document.xml: .//w:numPr` |
| Conditional table styling is currently partial. | planned | [word-table](office/word-table.docx) | `word/styles.xml: .//w:tblStylePr` |
| Floating table positioning is marked, not faithfully rendered. | planned | [word-conditional-floating](office/word-conditional-floating.docx) | `word/document.xml: .//w:tblpPr` |
| Multi-column section layout is currently approximate. | planned | [word-columns](office/word-columns.docx) | `word/document.xml: .//w:cols[@w:num="2"]` |
| Within-row/paragraph splitting is not implemented; expansion is tested. | planned | [word-long-tables](office/word-long-tables.docx) | `word/document.xml: .//w:tr (at least 3)` |
| Images/drawings are omitted with a visible marker. | planned | [word-images](office/word-images.docx) | `word/document.xml: .//w:drawing` |
| Text boxes are not presented as ordinary body text. | planned | [word-textbox](office/word-textbox.docx) | `word/document.xml: .//w:txbxContent` |
| Equations are marked as unrendered. | planned | [word-equations](office/word-equations.docx) | `word/document.xml: .//m:oMath` |
| Footnotes/endnotes and their references are marked. | planned | [word-notes](office/word-notes.docx) | `word/document.xml: .//w:footnoteReference`; `word/document.xml: .//w:endnoteReference` |
| Comments are retained with omission markers. | planned | [word-comments](office/word-comments.docx) | `word/document.xml: .//w:commentReference` |
| Tabs and inline breaks have explicit omission markers. | planned | [word-page-break](office/word-page-break.docx) | `word/document.xml: .//w:tab`; `word/document.xml: .//w:br` |
| Dedicated display is pending; editing is refused. | planned | [word-tracked](office/word-tracked.docx) | `word/document.xml: .//w:rPrChange` |
| Dedicated display is pending; editing is refused. | planned | [word-fields](office/word-fields.docx) | `word/document.xml: .//w:fldSimple` |
| Dedicated display is pending; editing is refused. | planned | [word-controls](office/word-controls.docx) | `word/document.xml: .//w:sdt` |
| Protected sources remain view-only. | planned | [word-protection](office/word-protection.docx) | `word/settings.xml: .//w:documentProtection` |
| Worksheet names/order, including empty sheets. | approximate | [excel-pivot](office/excel-pivot.xlsx) | `xl/workbook.xml: .//s:sheet (at least 4)` |
| Existing numeric/shared/inline/boolean cells; errors read-only. | approximate | [excel-errors-types](office/excel-errors-types.xlsx) | `xl/worksheets/*.xml: .//s:c[@t="b"]`; `xl/worksheets/*.xml: .//s:c[@t="e"]`; `xl/worksheets/*.xml: .//s:c[@t="s"]` |
| Inline/rich string values have a plain cell projection. | approximate | [excel-inline-rich-text](office/excel-inline-rich-text.xlsx) | `xl/worksheets/*.xml: .//s:c[@t="inlineStr"]/s:is` |
| Formula text is displayed read-only; no calculation. | approximate | [excel-cell-metadata](office/excel-cell-metadata.xlsx) | `xl/worksheets/*.xml: .//s:f` |
| Existing-cell grids page rows and columns. | approximate | [excel-wide-paging](office/excel-wide-paging.xlsx), [excel-wide-paging](office/excel-wide-paging.xlsx) | `xl/worksheets/*.xml: .//s:c[@r="AA1"]`; `xl/workbook.xml: .//s:fileVersion[@appName="xl"][@lastEdited="4"][@rupBuild="4505"]` |
| Dates remain serials; date/number display is planned. | planned | [excel-date-formats](office/excel-date-formats.xlsx) | `xl/styles.xml: .//s:numFmt`; `xl/worksheets/*.xml: .//s:c[@s]` |
| Merges are preserved but not rendered. | planned | [excel-styles](office/excel-styles.xlsx) | `xl/worksheets/*.xml: .//s:mergeCell` |
| Charts and pivots remain in the package, not the grid. | planned | [excel-pivot](office/excel-pivot.xlsx), [excel-extensions](office/excel-extensions.xlsx) | `xl/pivotTables/*.xml: .`; `xl/charts/chart*.xml: .` |
| Existing cell metadata and extension parts survive edits. | preserved | [excel-cell-metadata](office/excel-cell-metadata.xlsx) | `xl/worksheets/*.xml: .//s:c[@vm]` |
| Style/pivot extensions remain unchanged in native downloads. | preserved | [excel-extensions](office/excel-extensions.xlsx) | `xl/styles.xml: .//s:extLst`; `xl/pivotTables/*.xml: .//s:extLst` |
| 1904 date epoch and formats survive native saves. | preserved | [excel-1904-epoch](office/excel-1904-epoch.xlsx) | `xl/workbook.xml: .//s:workbookPr[@date1904="1"]` |
| Custom format IDs and definitions survive edits. | preserved | [excel-custom-formats](office/excel-custom-formats.xlsx) | `xl/styles.xml: .//s:numFmt` |
| Text-only slides follow presentation relationship order. | approximate | [powerpoint-slide-order](office/powerpoint-slide-order.pptx) | `ppt/presentation.xml: .//p:sldId (at least 2)` |
| Rich text/effects are preserved; richer layout is planned. | planned | [powerpoint-text-effects](office/powerpoint-text-effects.pptx) | `ppt/slides/slide*.xml: .//a:rPr[@b="1"]`; `ppt/slides/slide*.xml: .//a:effectLst` |
| Shapes/coordinates are preserved but not rendered. | planned | [powerpoint-shapes](office/powerpoint-shapes.pptx) | `ppt/slides/slide*.xml: .//p:sp`; `ppt/slides/slide*.xml: .//a:xfrm` |
| Images remain in downloads, not the text view. | planned | [powerpoint-image](office/powerpoint-image.pptx) | `ppt/slides/slide*.xml: .//p:pic` |
| Master/theme data stays unchanged. | planned | [powerpoint-shapes](office/powerpoint-shapes.pptx) | `ppt/theme/*.xml: .//a:themeElements` |
| Table geometry is preserved; text runs have a text-only projection. | planned | [powerpoint-table](office/powerpoint-table.pptx) | `ppt/slides/slide*.xml: .//a:tbl` |
| Header/footer contents are preserved and marked as unrendered. | planned | [word-headers-footers](office/word-headers-footers.docx) | `word/header*.xml: .`; `word/footer*.xml: .` |

The repeating-header fixture also has a native existing-text reflow probe in the
manifest. Its post-edit screenshot and assertions exercise continuation, repeated
header mirrors, oversized row expansion and unique field identity. Spreadsheet
views cover every sheet (including empty ones), plus a second row and column page.
Supplemental generated fixtures remain independent edge/preservation tests; they
are not counted as Office-authored files. New claims require a linked Office case
and reviewed semantic/visual expectations before being declared covered.
