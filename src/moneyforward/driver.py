from selenium import webdriver
import os

def get_remote_driver():
    options=webdriver.ChromeOptions()
    # options.add_argument("--headless")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-gpu")
    options.add_argument("--lang=ja-JP")
    options.add_argument("--disable-dev-shm-usage")
    options.add_experimental_option("prefs", {"download.default_directory": "/data/" })
    UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"
    options.add_argument("--user-agent=" + UA)
    driver = webdriver.Remote(
        command_executor=os.getenv("chromeAddr"),
        options=options
    )
    
    driver.implicitly_wait(10)
    return driver
