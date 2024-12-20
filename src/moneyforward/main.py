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

def main():
    global driver
    try:
        driver = driver.get_driver()
        run_scenario(driver=driver)
    except Exception as e:
        lg.error("failed to run fetch program", e, stack_info=True)
    finally:
        # ブラウザを閉じる
        driver.quit()

def run_scenario(driver):
    login()
    lg.info("login OK")

def login():
    url = "https://moneyforward.com/cf"  # for login page without account_selector
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
