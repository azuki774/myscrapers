from selenium import webdriver
import os

def get_remote_driver():
    options=webdriver.ChromeOptions()
    driver = webdriver.Remote(
        command_executor=os.getenv("chromeAddr"),
        options=options
    )
    
    driver.implicitly_wait(10)
    return driver
