import time
import uuid

import pymysql.cursors

# conn = pymysql.connect("localhost", "root", "hscxrzs1st", "test", charset='utf8')
db = pymysql.connect(user="root", passwd="123456", db="sync_es")
db.autocommit(True)
conn = db.cursor()


def insert_resource():
    for i in range(1000):
        now = int(time.time())
        name = uuid.uuid4()
        sql = "insert into `resource`(`name`, description, create_time, update_time, delete_time) values('%s', '%s', %d, %d, 0);" % (
        str(name), str(name), now, now)
        # print(sql)
        conn.execute(sql)
        for j in range(3):
            conn.execute(
                "insert into `resource_role`(user_id, resource_id, role_id, create_time, update_time, delete_time) values(%d, %d, %d, %d, %d, 0);" %
                (j + 1, conn.lastrowid, j + 1, now, now))


if __name__ == '__main__':
    insert_resource()
