# 열람실 좌석 현황 png

- 기본 json 형식

    ```json
    {
        "background": "readingroom_04.png",
        "width": 0,
        "height": 0,
        "seat-location": {
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

## json 추출 방법

- 웹 브라우저 실행 후 콘솔에서 아래 스크립트를 실행하여 추출.

```javascript
(function() {

  console.log();
})()
```
