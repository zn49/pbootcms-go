package admin

import "testing"

func TestSafeTableList(t *testing.T) {
	got := safeTableList([]string{"ay_user", "ay_user;drop", "ay_role", ""})
	if len(got) != 2 || got[0] != "ay_user" || got[1] != "ay_role" {
		t.Fatalf("%v", got)
	}
}

func TestPatchDatabaseTpl(t *testing.T) {
	in := `{php}$sum1 = 0; $sum2 = 0;{/php}
{php}
                                $size = round(($value->Data_length + $value->Index_length)/1024,2);
                                $sum1 += $size; 
                            {/php}
                            <td>{php}echo $size{/php} KB</td>
<td>
                            {if($value->Data_free)}
                                {php}
                                    $free = round($value->Data_free/1024,2);
                                    echo $free.' KB';
                                    $sum2 += $free; 
                                {/php}
                            {else}
                                -
                            {/if}
                            </td>
<td>[num] 张表</td>
<td>{php}echo $sum1{/php} KB</td>
<td>{php}echo $sum2?:0{/php} KB</td>`
	out := patchDatabaseTpl(in)
	if !containsAll(out, "[value->Size]", "[value->Free]", "{$db_sum_size}", "{$db_table_count}") {
		t.Fatalf("%s", out)
	}
	if containsAll(out, "{php}") {
		t.Fatalf("php left: %s", out)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !stringContains(s, p) {
			return false
		}
	}
	return true
}

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
