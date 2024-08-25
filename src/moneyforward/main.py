import driver
import money
import os
import sys
import time
from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.by import By
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
import logging
from pythonjsonlogger import jsonlogger

ROOTPAGE_URL = "https://moneyforward.com"

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
    global driver
    try:
        driver = driver.get_remote_driver()
        run_scenario(driver=driver)
    except Exception as e:
        lg.error("failed to run fetch program", e, stack_info=True)
    finally:
        # ブラウザを閉じる
        driver.quit()

def run_scenario(driver):
    # login
    try:
        html = money.login(driver)
    except Exception as e:
        lg.error("failed to login. maybe changing xpath: %s", e)
        driver.quit()
        sys.exit(1)
    lg.info("login ok")

    urls = os.getenv("urls").split(",")

    # Refresh Button
    money.move_page(driver, ROOTPAGE_URL)

    # refresh_xpaths に設定があれば、更新ボタンを押す
    try:
        refresh_xpaths = os.getenv("refresh_xpaths").split(",")
        for xpath in refresh_xpaths:
            try:
                money.press_from_xpath(driver, xpath)
                lg.info("press update button: %s", xpath)
                time.sleep(30)  # 反映されるように30sec待っておく
            except Exception as e:
                lg.warn("failed to press update button: %s", e)
                # update ボタンが押せなくとも終了しない
    except Exception as e:
        # update ボタンが押せななかったか、そもそも未設定の場合
        lg.warn("refresh button error or not set: %s", e)

    # download HTML
    for url in urls:
        try:
            html = money.get_from_url(driver, url)
            money.write_html(html, url)
            if (
                url == "https://moneyforward.com/cf"
            ):  # このページは先月分のデータも取っておく
                html = money.get_from_url_cf_lastmonth(driver)
                money.write_html(html, url + "_lastmonth")
        except Exception as e:
            lg.error("failed to get HTML: %s", e)

if __name__ == "__main__":
    main()
