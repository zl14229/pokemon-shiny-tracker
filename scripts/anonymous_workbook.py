"""Generate an anonymous XLSX to stdout; never reads a user's workbook."""
import io
import zipfile
from xml.sax.saxutils import escape
import sys


def workbook():
    rows = [["编号", "日期", "球种", "备注"], ["25", "2020-02", "匿名球", "T012 anonymous fixture"], ["25", "invalid", "", "invalid row"], ["25", "2020-02", "匿名球", "distinct possible individual"]]
    sheet = ["<worksheet><sheetData>"]
    for n, row in enumerate(rows, 1):
        sheet.append(f'<row r="{n}">')
        for index, text in enumerate(row):
            sheet.append(f'<c r="{chr(65 + index)}{n}" t="inlineStr"><is><t>{escape(text)}</t></is></c>')
        sheet.append("</row>")
    sheet.append("</sheetData></worksheet>")
    parts = {
        "xl/workbook.xml": '<workbook xmlns:r="urn:r"><sheets><sheet name="匿名样本" r:id="r1"/></sheets></workbook>',
        "xl/_rels/workbook.xml.rels": '<Relationships><Relationship Id="r1" Target="worksheets/sheet1.xml"/></Relationships>',
        "xl/worksheets/sheet1.xml": "".join(sheet),
    }
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as archive:
        for name, text in parts.items():
            archive.writestr(name, text)
    return buf.getvalue()


if __name__ == "__main__":
    sys.stdout.buffer.write(workbook())
