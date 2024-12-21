import driver
import os
import datetime
import time
import logging
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
    row_csv_data = download_csv_from_page()
    lg.info("download csv OK")

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

def convert_csv_data(fetch_data, lastmonth):
    """
    download_csv_from_page() で取得したデータを、MoneyForward公式のCSV形式に変換する

    差異は下記
    - 計算対象は無条件で1にする
    - ID 部分は取得不可なので、空文字にする
    - 振替欄も正しく入らない（空文字）
    - ただし、文字コードは UTF8 のままにする（公式はSJIS）

    ['', '12/09(月)', '物販', '-110', 'モバイルSuica', '未分類', '未分類', '', '', '']
    - > "1","2024/12/09","物販","-110","モバイルSuica",”未分類","未分類","","",""
    """
    pass

def convert_date_field(date_text, now_date, lastmonth):
    """
    今年 .. 2024年とする
    12/09(月) -> 2024/12/09 に変換
    ただし、lastmonth = True （先月のデータ）の場合は、
    12/09（＊）-> 2023/12/09 に変換する（2024/12/09でなく）
    """
    # now_date  # now_date.date.today() or datetime.date(2020, 3, 22)
    year = now_date.year
    month = now_date.month
    day = now_date.day

    text_month = date_text[0:2]
    if (lastmonth == True) and (text_month == "12"):
        return str(year - 1) +  "/" + date_text[0:5]
    return str(year) + "/" + date_text[0:5]

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
