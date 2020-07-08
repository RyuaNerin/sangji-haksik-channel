# 버스 노선표

1. 아래 두 파일을 수정해주세요
    |파일명|설명|
    |----|----|
    |[static/bus-in.md](static/bus-in.md)|시내 버스 노선표|
    |[static/bus-out.md](static/bus-out.md)|시외 버스 노선표|

2. html 파일을 생성해주세요
    - [pandoc](https://pandoc.org/MANUAL.html) 이 필요합니다. (작성 기준 `v2.9.2` 사용)

        ```shell
        make
        ```

        혹은

        ```shell
        pandoc --template=template.htm --table-of-contents 'static/bus-in.md'  > 'static/bus-in.htm'
        pandoc --template=template.htm --table-of-contents 'static/bus-out.md' > 'static/bus-out.htm'
        ```

3. 변경사항을 커밋한 후 **Pull Request** 넣어주세요.
