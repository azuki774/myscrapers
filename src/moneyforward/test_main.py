import main
import datetime

def test_convert_date_field():
    test_data = [
        {
            "date_text": "03/12（＊）",
            "now_date": datetime.date(2024, 3, 14),
            "lastmonth": False,
            "want": "2024/03/12"
        },
        {
            "date_text": "02/12（＊）",
            "now_date": datetime.date(2024, 3, 14),
            "lastmonth": True,
            "want": "2024/02/12"
        },
        {
            "date_text": "12/12（＊）",
            "now_date": datetime.date(2024, 12, 14),
            "lastmonth": False,
            "want": "2024/12/12"
        },
                {
            "date_text": "12/17（＊）",
            "now_date": datetime.date(2024, 1, 3),
            "lastmonth": True,
            "want": "2023/12/17"
        },
    ]
    for i, t in enumerate(test_data):
        assert main.convert_date_field(t["date_text"], t["lastmonth"], t["now_date"]) == t["want"]

def test_convert_csv_data():
    test_data = [
        {
            "input": ['', '12/09(月)', '物販', '-110', 'モバイルSuica', '未分類', '未分類', '', '', ''],
            "lastmonth": False,
            "now_date": datetime.date(2024, 1, 23),
            "want": '"1","2024/12/09","物販","-110","モバイルSuica","未分類","未分類","","",""',
        }
    ]

    for i, t in enumerate(test_data):
        print(main.convert_csv_data(t["input"], t["lastmonth"], t["now_date"]).encode())
        print(t["want"].encode())
        assert main.convert_csv_data(t["input"], t["lastmonth"], t["now_date"]) == t["want"]
