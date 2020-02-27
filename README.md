# 상지대 식단표

- 카카오 플러스친구 [상지대 식단표](http://pf.kakao.com/_xbkxdyT) 의 Webhook 서버입니다
- 이 서비스는 상지대학교의 공식 서비스가 아닙니다.
- 이 레포지토리는 [GNU General Public License version 3.0 (GPLv3)](LICENSE.txt) 하에 배포됩니다.

## 참조 사항

- 이 서비스는 [**golang 용 카카오 i 오픈빌더 API 라이브러리** `go-kakaoskill`](https://github.com/RyuaNerin/go-kakaoskill) 로 제작되었습니다.

- 식단표는 5분마다 갱신됩니다
    - `menu.go` 파일 `UpdatePeriod` 상수