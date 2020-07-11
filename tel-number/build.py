# -*- coding:utf-8 -*-

import csv
import re

with open("build.csv", "w", encoding="utf-8", newline="") as _fw:
    _fw.truncate(0)
    _fw.write("\ufeff")

    writer = csv.writer(_fw)
    writer.writerow(
        [
            "FAQ_No",
            "Category1",
            "Category2",
            "Category3",
            "Category4",
            "Category5",
            "Question",
            "Answer",
            "Landing URL",
            "Image info (URL)",
        ]
    )

    row_number = 0

    ###########################################################################

    with open("department.csv", "r", encoding="utf-8") as _fr:
        f = True
        for row in csv.reader(_fr, delimiter="\t"):
            if f:
                f = False
                continue

            category2 = row[0]
            category3 = row[1]
            question_list = [row[1]]
            answer = "<{0}>\n{1}\n{2}".format(category2, category3, row[2])
            url = row[3]

            question_list.extend(row[4:])

            tm = re.match(r"^\d\d\d\-(\d\d\d\-(\d\d\d\d))$", row[2])
            if tm is not None:
                question_list.extend([row[2], tm[1], tm[2]])

            for question in [
                x for x in set(question_list) if x is not None and len(x) > 0
            ]:
                row_number += 1
                writer.writerow(
                    [
                        row_number,
                        "과사무실",
                        category2,
                        category3,
                        None,
                        None,
                        question,
                        answer,
                        url,
                        None,
                    ]
                )

    ###########################################################################

    with open("infomation.csv", "r", encoding="utf-8") as _fr:
        f = True
        for row in csv.reader(_fr, delimiter="\t"):
            if f:
                f = False
                continue

            if len([p for p in row[2:7] if p is not None]) == 0:
                continue

            category2 = row[0]
            category3 = row[1]
            question_list = [row[1]]
            answer = "<{0}>\n{1}\n{2}\n\n{3}".format(
                category2,
                category3,
                "\n".join(
                    [tel for tel in row[3:7] if tel is not None and len(tel) > 0]
                ).strip(),
                row[2],
            ).strip()
            url = row[8]

            question_list.extend(row[9:])

            for tel in row[3:7]:
                tm = re.match(r"^\d\d\d\-(\d\d\d\-(\d\d\d\d))$", tel)
                if tm is not None:
                    question_list.extend([tel, tm[1], tm[2]])

            for question in [
                x for x in set(question_list) if x is not None and len(x) > 0
            ]:
                row_number += 1
                writer.writerow(
                    [
                        row_number,
                        "부서 및 시설물",
                        category2,
                        category3,
                        None,
                        None,
                        question,
                        answer,
                        url,
                        None,
                    ]
                )
