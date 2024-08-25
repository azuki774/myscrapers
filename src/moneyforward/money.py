import time
import datetime as dt
import os
from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.by import By
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
import logging
import json
from pythonjsonlogger import jsonlogger

lg = logging.getLogger(__name__)
lg.setLevel(logging.DEBUG)
h = logging.StreamHandler()
h.setLevel(logging.DEBUG)
json_fmt = jsonlogger.JsonFormatter(
    fmt="%(asctime)s %(levelname)s %(name)s %(message)s", json_ensure_ascii=False
)
h.setFormatter(json_fmt)
lg.addHandler(h)

SAVE_DIR = "/data"


def login(driver):
    url = "https://moneyforward.com/cf"  # for login page without account_selector
    driver.get(url)

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

    rl = "https://moneyforward.com/"
    driver.get(url)
    html = driver.page_source.encode("utf-8")
    return html


def get_from_url(driver, url):
    wait = WebDriverWait(driver=driver, timeout=30)

    lg.info("fetch url: " + url)
    driver.get(url)
    wait.until(EC.presence_of_all_elements_located)
    html = driver.page_source.encode("utf-8")
    return html


def get_from_url_cf_lastmonth(driver):
    # cf ページの last_month を取得して書き出す関数
    wait = WebDriverWait(driver=driver, timeout=30)

    url = "https://moneyforward.com/cf"
    lg.info("fetch url: " + url)
    driver.get(url)
    wait.until(EC.presence_of_all_elements_located)

    # lastmonth_button button
    lastmonth_button = driver.find_element(
        by=By.XPATH,
        value="/html/body/div[1]/div[3]/div/div/section/div[2]/button[1]",
    )
    lastmonth_button.click()
    time.sleep(2)
    html = driver.page_source.encode("utf-8")
    return html


def move_page(driver, url):
    wait = WebDriverWait(driver=driver, timeout=30)
    lg.info("move page url: " + url)
    driver.get(url)
    return


def press_from_xpath(driver, xpath):
    """
    指定したxpathのリンクを押す
    ページはすでに遷移済にしておくこと
    """
    xpath_link = driver.find_element(
        by=By.XPATH,
        value=xpath,
    )
    xpath_link.click()
    return


def get_status(driver, xpaths):
    """
    /html/body/div[1]/div[3]/div[1]/div[1]/div[2]/div[1]/div/section[3]/ul/li[3]/ul[2]/li[3]/a[2] <- key: 「更新」リンクのxpath
    たちから
    /html/body/div[1]/div[3]/div[1]/div[1]/div[2]/div[1]/div/section[3]/ul/li[3]/div/a[1] : 名前
    /html/body/div[1]/div[3]/div[1]/div[1]/div[2]/div[1]/div/section[3]/ul/li[3]/div/div : 取得日
    /html/body/div[1]/div[3]/div[1]/div[1]/div[2]/div[1]/div/section[3]/ul/li[3]/ul[2]/li[1] : 同期ステータス
    を取得して、リストで返す
    """
    move_page(driver, "https://moneyforward.com")
    ret_f = {}

    for xpath in xpaths:
        base_xpath_list = xpath.split("/")[0:-3]
        base_xpath = "/".join(
            base_xpath_list
        )  # /html/body/div[1]/div[3]/div[1]/div[1]/div[2]/div[1]/div/section[3]/ul/li[3]
        name_xpath = base_xpath + "/div/a[1]"
        syncday_xpath = base_xpath + "/div/div"
        sync_status_xpath = base_xpath + "/ul[2]/li[1]"

        name = driver.find_element(by=By.XPATH, value=name_xpath).get_attribute(
            "textContent"
        )

        syncday = driver.find_element(by=By.XPATH, value=syncday_xpath).get_attribute(
            "textContent"
        )

        sync_status = driver.find_element(
            by=By.XPATH, value=sync_status_xpath
        ).get_attribute("textContent")

        ret_f[name] = {"sync_day": syncday, "sync_status": sync_status}

    ret_f_json = json.dumps(ret_f, ensure_ascii=False)
    return ret_f_json


def write_html(html, url):
    today = dt.date.today()  # 出力：datetime.date(2020, 3, 22)
    yyyymm = "{0:%Y%m}".format(today)  # 202003
    yyyymmdd = "{0:%Y%m%d}".format(today)  # 20200322
    os.makedirs(SAVE_DIR + yyyymm, exist_ok=True)
    os.makedirs(SAVE_DIR + yyyymmdd, exist_ok=True)
    path_w = SAVE_DIR + "/" + yyyymm + "/" + yyyymmdd + "/" + os.path.basename(url) + ".html"
    with open(path_w, mode="w") as f:
        f.write(html.decode("utf-8"))
    lg.info("write ok")
    return
