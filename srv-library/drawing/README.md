# 열람실 좌석 현황 png

## 폰트 생성 방법

- `맑은 고딕`
- subset : `0123456789년월일화수목금토요시분초오전후기준`

```shell
> pyftsubset malgun.ttf --unicodes="U+30,U+31,U+32,U+33,U+34,U+35,U+36,U+37,U+38,U+39,U+B144,U+C6D4,U+C77C,U+D654,U+C218,U+BAA9,U+AE08,U+D1A0,U+C694,U+C2DC,U+BD84,U+CD08,U+C624,U+C804,U+D6C4,U+AE30,U+C900" --output-file="malgun-subset.ttf"
> pyftsubset malgunbd.ttf --unicodes="U+30,U+31,U+32,U+33,U+34,U+35,U+36,U+37,U+38,U+39,U+B144,U+C6D4,U+C77C,U+D654,U+C218,U+BAA9,U+AE08,U+D1A0,U+C694,U+C2DC,U+BD84,U+CD08,U+C624,U+C804,U+D6C4,U+AE30,U+C900" --output-file="malgunbd-subset.ttf"
```

## 좌표 데이터 생성 방법

- 기본 json 형식

    ```json
    {
        "width" : 0,
        "height" : 0,
        "background": "readingroom_04.png",
        "location": {
            "background" : {
                "x": 0,
                "y": 0,
            },
            "0": {
                "x": 0,
                "y": 0,
                "border-top": {
                    "color": "#d53212",
                    "thickness": 1
                },
                "border-left": {
                    "color": "#d53212",
                    "thickness": 1
                },
                "border-right": {
                    "color": "#d53212",
                    "thickness": 1
                },
                "border-bottom": {
                    "color": "#d53212",
                    "thickness": 1
                }
            }
        }
    }
    ```

- 웹 브라우저 실행 후 콘솔에서 [이 스크립트](create-json.js)를 실행하여 생성.
