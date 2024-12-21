import driver
import os
import datetime
import time
import logging
import datetime
from pythonjsonlogger import jsonlogger
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from bs4 import BeautifulSoup

lg = logging.getLogger(__name__)
lg.setLevel(logging.DEBUG)
h = logging.StreamHandler()
h.setLevel(logging.DEBUG)
json_fmt = jsonlogger.JsonFormatter(
    fmt="%(asctime)s %(levelname)s %(name)s %(message)s", json_ensure_ascii=False
)
h.setFormatter(json_fmt)
lg.addHandler(h)

SBI_USER = os.getenv("user")
SBI_PASS = os.getenv("pass")
SAVE_DIR = "/data"
CF_PAGE='https://moneyforward.com/cf'

def main():
    global driver
    try:
        driver = driver.get_driver()
        run_scenario()
    except Exception as e:
        lg.error("failed to run fetch program", e, stack_info=True)
    finally:
        # ブラウザを閉じる
        driver.quit()

def run_scenario():
    login()
    lg.info("login OK")
    # this month download
    row_csv_data = download_csv_from_page(False)
    lg.info("download record OK")

    print(row_csv_data)
    csv_text = []
    for rc in row_csv_data:
        # 1行ごとの文字列に変換
        row_csv_text = convert_csv_data(rc, False, None)
        csv_text.append(row_csv_text)
    
    lg.info("parse record OK")
    write_csv(csv_text, "/data/cf.csv")
    lg.info("write csv OK")

def login():
    url = CF_PAGE  # for login page without account_selector
    driver.get(url)
    lg.info("move Login page")

    login_id = driver.find_element(
        by=By.XPATH,
        value="/html/body/main/div/div/div[2]/div/section/div/form/div/div/input",
    )
    login_id.send_keys(os.getenv("user"))

    lg.info("input login")

    email_button = driver.find_element(
        by=By.XPATH,
        value="/html/body/main/div/div/div[2]/div/section/div/form/div/button",
    )
    email_button.click()

    lg.info("input email button")

    try:
        password_form = driver.find_element(
            by=By.XPATH,
            value="/html/body/main/div/div/div[2]/div/section/div/form/div/div[2]/input",
        )
        password_form.send_keys(os.getenv("pass"))
        lg.info("input password")

        login_button = driver.find_element(
            by=By.XPATH,
            value="/html/body/main/div/div/div[2]/div/section/div/form/div/button",
        )
        login_button.click()
        lg.info("input login_button")

    except Exception as e:
        lg.info("maybe already login. skipped.")

    url = "https://moneyforward.com/"
    driver.get(url)
    html = driver.page_source.encode("utf-8")
    return html

# 今開いているcfページ
def download_csv_from_page(lastmonth):
    # ページソース取得
    url = CF_PAGE
    driver.get(url)
    lg.info("move cf page")
    html = driver.page_source
    soup = BeautifulSoup(html, "html.parser")
    table = soup.find(id="cf-detail-table")
    tr_list = table.find_all("tr")
    fetch_data = []
    for i, tr in enumerate(tr_list):
        row_data = []
        td_list = tr.find_all("td")
        for j, td in enumerate(td_list):
            row_data.append(td.get_text().strip())
        if len(row_data) > 0:
            # 空行以外を挿入
            fetch_data.append(row_data)
    return fetch_data

def convert_csv_data(fetch_data, lastmonth, now_date):
    """
    download_csv_from_page() で取得したデータの1行を、MoneyForward公式のCSV形式に変換する

    差異は下記
    - 計算対象は無条件で1にする
    - ID 部分は取得不可なので、空文字にする
    - 振替欄も正しく入らない（空文字）
    - ただし、文字コードは UTF8 のままにする（公式はSJIS）

    ['', '12/09(月)', '物販', '-110', 'モバイルSuica', '未分類', '未分類', '', '', '']
    - > "1","2024/12/09","物販","-110","モバイルSuica","未分類","未分類","","",""
    """
    res_text = '"{0}","{1}","{2}","{3}","{4}","{5}","{6}","{7}","{8}","{9}"'.format(
        1, # 固定値
        convert_date_field(fetch_data[1], lastmonth, now_date),
        fetch_data[2].split('\n')[0], # 最初の改行以降は消す
        fetch_data[3].split('\n')[0],
        fetch_data[4].split('\n')[0],
        fetch_data[5].split('\n')[0],
        fetch_data[6].split('\n')[0],
        fetch_data[7].split('\n')[0],
        fetch_data[8].split('\n')[0],
        fetch_data[9].split('\n')[0],
    )
    return res_text

def convert_date_field(date_text, lastmonth, now_date):
    """
    今年 .. 2024年とする
    12/09(月) -> 2024/12/09 に変換
    ただし、lastmonth = True （先月のデータ）の場合は、
    12/09（＊）-> 2023/12/09 に変換する（2024/12/09でなく）
    """
    if now_date == None:
        # now_date に指定がなければ現在時刻
        now_date = datetime.date.today()

    year = now_date.year
    month = now_date.month
    day = now_date.day

    text_month = date_text[0:2]
    if (lastmonth == True) and (text_month == "12"):
        return str(year - 1) +  "/" + date_text[0:5]
    return str(year) + "/" + date_text[0:5]

def write_csv(csv_data, path_w):
    with open(path_w, mode='w') as f:
        # ヘッダ書き込み
        f.write('"計算対象","日付","内容","金額（円）","保有金融機関","大項目","中項目","メモ","振替","ID"\n')
        for d in csv_data:
            f.write(d + '\n')

def download_detailCSV(yyyymm):
    filename = yyyymm + ".csv"
    lg.info("download start")
    driver.get('https://moneyforward.com/cf/csv?from=' + "2024" + '%2F' + "10" + '%2F01&month=' + "2024" + '&year=' + "10")
    time.sleep(10)
    html = driver.page_source.encode("utf-8")
    print(html)

    lg.info("download complete")
    lg.info("output to " + filename)

# ディレクトリ作成とファイル名取得する
def get_file_path(index, lastmonth):
    today = datetime.date.today()  # 出力：datetime.date(2020, 3, 22)
    yyyymm = "{0:%Y%m}".format(today)  # 202003

    if lastmonth == False:
        # 今月
        filepath = SAVE_DIR + "/" + yyyymm + ".csv"
    return filepath


if __name__ == "__main__":
    main()
