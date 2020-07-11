# -*- coding:utf-8 -*-

import csv
import re

with open("build.csv", "w", encoding="utf-8") as _fw:
    _fw.truncate(0)

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

    with open("department.csv", "r", encoding="utf-8") as _fr:
        keyword_dup = set()

        for row in csv.reader(_fr, delimiter="\t"):
            if row[0] == "대학":
                continue

            category2 = row[0]
            category3 = row[1]
            question_list = [row[1]]
            answer = "{0}\n{1}\n{2}".format(category2, category3, row[2])
            url = row[3]

            question_list.extend([keyword for keyword in row[4:] if len(keyword) > 0])

            for question in set(question_list):
                if question in keyword_dup:
                    raise Exception("Keyword {} is Duplicated".format(question))
                keyword_dup.add(question)

            tm = re.match(r"^\d\d\d\-(\d\d\d)\-(\d\d\d\d)$", row[2])
            if tm is not None:
                question_list.extend(
                    [row[2], "{0}-{1}".format(tm[1], tm[2]), "{0}".format(tm[2]),]
                )

            for question in set(question_list):
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

