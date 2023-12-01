import unittest
import unit_db_test.testcase as dbtest

class CheckIntegrityOfDB(dbtest.DBintegrityTest):
    db_config_file = ".env"

    def test_num_validators_equals_sum_of_val_states(self):
        """ Columns f_num_slashed, f_num_active, f_num_exit, f_num_in_activation should sum up to f_num_vals"""
        sql_query = """
        SELECT *
        FROM t_epoch_metrics_summary
        WHERE (f_num_slashed + f_num_active + f_num_exit + f_num_in_activation) != f_num_vals;
        """
        df = self.db.get_df_from_sql_query(sql_query)
        self.assertNoRows(df)

if __name__ == '__main__':
    unittest.main()


