/**
 * Build an .xlsx workbook from row objects and trigger a browser download.
 * Filename must come from the API (fail closed - never invent names).
 */
export async function downloadRowsAsXlsx(
  rows: Array<Record<string, unknown>>,
  filename: string,
  sheetName = "Sheet1",
): Promise<void> {
  const name = filename.trim();
  if (!name) {
    throw new Error("");
  }
  const ExcelJS = (await import("exceljs")).default;
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet(sheetName.slice(0, 31) || "Sheet1");

  const keys = collectKeys(rows);
  if (keys.length === 0) {
    sheet.addRow(["(empty)"]);
  } else {
    sheet.addRow(keys);
    const header = sheet.getRow(1);
    header.font = { bold: true };
    for (const row of rows) {
      sheet.addRow(keys.map((k) => cellValue(row[k])));
    }
    for (const col of sheet.columns) {
      let max = 10;
      col.eachCell?.({ includeEmpty: true }, (cell) => {
        const len = String(cell.value ?? "").length;
        if (len > max) max = Math.min(len, 60);
      });
      col.width = max + 2;
    }
  }

  const buffer = await workbook.xlsx.writeBuffer();
  const blob = new Blob([buffer], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name.endsWith(".xlsx") ? name : name.replace(/\.json$/i, ".xlsx");
  a.click();
  URL.revokeObjectURL(url);
}

function collectKeys(rows: Array<Record<string, unknown>>): string[] {
  const seen = new Set<string>();
  const keys: string[] = [];
  for (const row of rows) {
    for (const k of Object.keys(row)) {
      if (!seen.has(k)) {
        seen.add(k);
        keys.push(k);
      }
    }
  }
  return keys;
}

function cellValue(v: unknown): string | number | boolean | null {
  if (v == null) return null;
  if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") return v;
  if (typeof v === "object") {
    try {
      return JSON.stringify(v);
    } catch {
      return String(v);
    }
  }
  return String(v);
}
