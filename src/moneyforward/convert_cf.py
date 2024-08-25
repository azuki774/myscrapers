import sys
import logging
from pythonjsonlogger import jsonlogger

"""
fetcher 本体で取得したファイルからCSV形式に抽出
cf.html -> cf.csv
cf_lastmonth.html -> cf_lastmonth.csv
"""

lg = logging.getLogger(__name__)
lg.setLevel(logging.DEBUG)
h = logging.StreamHandler()
h.setLevel(logging.DEBUG)
json_fmt = jsonlogger.JsonFormatter(
    fmt="%(asctime)s %(levelname)s %(name)s %(message)s", json_ensure_ascii=False
)
h.setFormatter(json_fmt)
lg.addHandler(h)


def main():
    args = sys.argv
    if len(args) != 2:
        lg.error("missing args")
        sys.exit(1)
    html_filedir = args[1]
    


if __name__ == "__main__":
    main()
