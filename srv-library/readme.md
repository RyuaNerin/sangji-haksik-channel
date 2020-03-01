# Template 작업 요령

- `2020-03-02` 기준 다음의 과정으로 Template 를 작성하였습니다.

1. `div.seat_table_wrap` 선택 후 HTML 복사
2. HTML 포메팅
3. 주석 제거
    |Regex|Replace|
    |-|-|
    |`<!--.*-->`||
4. 빈 라인 제거
    |Regex|Replace|
    |-|-|
    |`^[ \t]*\n`||
5. LineBreak 된 처리
    |Regex|Replace|
    |-|-|
    |`\n[ \t]*" *>`|`">`|
    |`(style="[^"]+)\n +([^"]+")`|`$1$2`|
6. 좌석 내용물 비우기
    |Regex|Replace|
    |-|-|
    |`<td id="[^"]+"`|`<td`|
    |`onclick="[^"]+" `||
    |`<input .+>`||
    |`<ul( [^>]+)?>((?!</ul>).\|\n)*</ul>`||
    |`using_seat`||
    |`<span class="seat_type_[^"]+"( [^>]+)?>((?!</span>).\|\n)*</span>`||
    |`<script [^>]+>((?!</script>).\|\n)*</script>`||
    |`cursor:pointer;`||
8. 빈 라인 제거
    |Regex|Replace|
    |-|-|
    |`^[ \t]*\n`||
7. Template 넣기
    |Regex|Replace|
    |-|-|
    |`<span class="seat_num">0*(\d+)</span>`|`{{ template "seat.tmpl.htm" index .Seat $1 }}`|
    - Header : `{{ template "header.tmpl.htm" . }}`
    - Footer :`{{ template "footer.tmpl.htm" . }}`
9. 리소스 복사 및 수정