import driver
import os
import datetime
import time
import logging
import json
import csv
import time
import argparse
import s3
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
CF_FILENAME="cf.csv"
CF_FILENAME_LASTMONTH="cf_lastmonth.csv"
CF_PAGE='https://moneyforward.com/cf'
ACCOUNTS_PAGE="https://moneyforward.com/accounts"

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--s3-upload", help="optional", action="store_true") # s3 upload機能の有効化フラグ
    args = parser.parse_args()
    global driver
    try:
        driver = driver.get_driver()
        run_scenario()
        if args.s3_upload:
            # s3 upload 機能フラグが有効なとき
            lg.info("s3 upload start")
            s3.upload_file(SAVE_DIR + "/" + CF_FILENAME)
            s3.upload_file(SAVE_DIR + "/" + CF_FILENAME_LASTMONTH)
            lg.info("s3 upload complete")
    except Exception as e:
        lg.error("failed to run fetch program", e, stack_info=True)
    finally:
        # ブラウザを閉じる
        driver.quit()

def run_scenario():
    cookies = load_cookies_from_json('/data/cookie.json')
    for cookie in cookies:
        driver.add_cookie(cookie)
    
    lg.info("load cookie OK")

    url = CF_PAGE # 1回入れないとうまくページ遷移しないので入れる
    driver.get(url)
    lg.info("move cf page")

    # update_accounts()

    # 「今月」ボタンを押す
    lg.info("press this month button")

    # this month download
    row_csv_data = download_csv_from_page(False)
    lg.info("download record OK")

    csv_text = []
    for rc in row_csv_data:
        # 1行ごとの文字列に変換
        row_csv_text = convert_csv_data(rc, False, None)
        csv_text.append(row_csv_text)
    
    lg.info("parse record OK")
    write_csv(csv_text, SAVE_DIR + "/" + CF_FILENAME)
    lg.info("write csv OK")

    # lastmonth ページを表示
    press_lastmonth_btn()
    lg.info("lastmonth page move ok")
    row_csv_data = download_csv_from_page(True)
    lg.info("download record(lastmonth) OK")

    csv_text = []
    for rc in row_csv_data:
        # 1行ごとの文字列に変換
        row_csv_text = convert_csv_data(rc, True, None)
        csv_text.append(row_csv_text)
    
    lg.info("parse record(lastmonth) OK")
    write_csv(csv_text, SAVE_DIR + "/" + CF_FILENAME_LASTMONTH)
    lg.info("write csv OK")

    lg.info("converting UTF-8 -> SJIS")
    utf8tosjis(SAVE_DIR + "/" + CF_FILENAME)
    utf8tosjis(SAVE_DIR + "/" + CF_FILENAME_LASTMONTH)
    lg.info("converting UTF-8 -> SJIS OK")

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

def update_accounts():
    url = ACCOUNTS_PAGE  # for login page without account_selector
    driver.get(url)
    lg.info("move accounts page")

    # 「金融機関からのデータ一括更新」ボタンを押す
    update_btn = driver.find_element(
        by=By.XPATH,
        value="/html/body/div[1]/div[2]/div[1]/div/div/div/section/p[2]/a",
    )
    update_btn.click()
    lg.info("press update button. wait 60sec")
    time.sleep(60) # 取得待ち

def load_cookies_from_json(filepath):
    url = CF_PAGE  # for login page without account_selector
    driver.get(url)
    lg.info("move Login page") # Cookie を設定するには一度そのドメインにログインする必要がある
    time.sleep(10)

    # Loads cookies from a JSON file and formats them for Selenium.
    cookies = []
    try:
        with open(filepath, 'r') as f:
            raw_cookies = json.load(f)

        for cookie in raw_cookies:
            # Format for Selenium, mapping keys like 'expirationDate' to 'expiry'
            selenium_cookie = {
                'name': cookie.get('name'),
                'value': cookie.get('value'),
                'domain': cookie.get('domain'),
                'path': cookie.get('path'),
                'secure': cookie.get('secure', False),
                'httpOnly': cookie.get('httpOnly', False),
            }
            # Add expiry if 'expirationDate' exists (should be Unix timestamp)
            if 'expirationDate' in cookie:
                try:
                    # Convert to integer Unix timestamp (seconds)
                    expiry_ts = int(cookie['expirationDate'])
                    # Optionally skip expired cookies (add_cookie might handle this)
                    # if expiry_ts < time.time():
                    #     continue
                    selenium_cookie['expiry'] = expiry_ts
                except (ValueError, TypeError):
                    pass # Ignore if conversion fails

            # Add sameSite attribute if present and valid
            if 'sameSite' in cookie and cookie['sameSite'] in ['Strict', 'Lax', 'None', 'no_restriction', 'lax', 'strict']:
                 # Selenium expects 'Strict', 'Lax', or 'None'
                 ss_val = cookie['sameSite'].capitalize()
                 if ss_val == 'No_restriction': ss_val = 'None' # Map common value from extensions
                 if ss_val in ['Strict', 'Lax', 'None']:
                    selenium_cookie['sameSite'] = ss_val

            # Check for required keys before adding
            if selenium_cookie.get('name') and selenium_cookie.get('value') and selenium_cookie.get('domain'):
                cookies.append(selenium_cookie)
            else:
                print(f"Warning: Skipping cookie with missing required keys: {cookie}")

    except FileNotFoundError:
        print(f"Error: Cookie file not found: {filepath}")
    except json.JSONDecodeError:
        print(f"Error: Invalid JSON format in file: {filepath}")
    except Exception as e:
        print(f"Error: An error occurred while loading the cookie file: {e}")

    return cookies


def download_csv_from_page(lastmonth):
    # 今開いているcfページ
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

def press_nowmonth_btn():
    # /cf ページにある「今月」ボタンを押す
    now_btn = driver.find_element(
        by=By.XPATH,
        value="/html/body/div[1]/div[2]/div/div/div/section/section/div[2]/div/div/div[1]/div/div[4]/span",
    )
    now_btn.click()
    time.sleep(5) # 画面遷移待ち

def press_lastmonth_btn():
    # 先月に移動する[<]ボタンを押す
    lastmonth_btn = driver.find_element(
        by=By.XPATH,
        value="/html/body/div[1]/div[2]/div/div/section/div[2]/button[1]",
    )
    lastmonth_btn.click()
    time.sleep(5) # 画面遷移待ち

def utf8tosjis(filename):
    """
    utf8 で書き出された CSV を SJIS に変換する
    """
    data = []
    try:
        with open(filename, 'r', encoding='utf-8', newline='') as infile:
            reader = csv.reader(infile)
            # 全ての行をリストに読み込む
            data = list(reader)
        with open(filename, 'w', encoding='shift_jis', newline='', errors='replace') as outfile:
            writer = csv.writer(outfile)
            # メモリに格納したデータを全て書き出す
            writer.writerows(data)
        lg.info(f"converted'{filename}' to Shift-JIS")
    except FileNotFoundError:
        lg.error(f"'{filename}' is not found")
    except Exception as e:
        lg.error(f"error occurred while converting '{filename}': {e}")

if __name__ == "__main__":
    main()
