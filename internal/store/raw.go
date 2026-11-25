package store

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (d *DB) queryInt(q string, args ...any) (int, error) {
	rows, err := d.SQL.Query(d.rebind(q), args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int
	if rows.Next() {
		if err := rows.Scan(&n); err != nil {
			return 0, err
		}
	}
	return n, rows.Err()
}

// Maps runs a raw query. Used for runtime diy tables Ent does not model.
func (d *DB) Maps(q string, args ...any) ([]map[string]string, error) {
	return d.queryMaps(q, args...)
}

func (d *DB) queryMaps(q string, args ...any) ([]map[string]string, error) {
	rows, err := d.SQL.Query(d.rebind(q), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]string
	for rows.Next() {
		raw := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range raw {
			ptr[i] = &raw[i]
		}
		if err := rows.Scan(ptr...); err != nil {
			return nil, err
		}
		m := make(map[string]string, len(cols))
		for i, c := range cols {
			m[c] = cellString(raw[i])
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) execRaw(q string, args ...any) error {
	_, err := d.SQL.Exec(d.rebind(q), args...)
	return err
}

// rebind turns ? into $n for PostgreSQL. Ent does this for its own queries.
func (d *DB) rebind(q string) string {
	if d.Driver != "pgsql" {
		return q
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(q); i++ {
		if q[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(q[i])
	}
	return b.String()
}

// Insert writes a runtime table (ay_diy_*). Fixed tables go through Ent.
func (d *DB) Insert(table string, cols map[string]any) error {
	if !Ident(table) {
		return fmt.Errorf("非法表名")
	}
	if len(cols) == 0 {
		return fmt.Errorf("无字段")
	}
	keys := make([]string, 0, len(cols))
	for k := range cols {
		if !Ident(k) {
			return fmt.Errorf("非法字段 %q", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tq, err := d.QuoteIdent(table)
	if err != nil {
		return err
	}
	names := make([]string, len(keys))
	ph := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, k := range keys {
		cq, err := d.QuoteIdent(k)
		if err != nil {
			return err
		}
		names[i] = cq
		ph[i] = "?"
		args[i] = cols[k]
	}
	return d.execRaw("INSERT INTO "+tq+" ("+strings.Join(names, ",")+") VALUES ("+strings.Join(ph, ",")+")", args...)
}

// ExtByContent loads ay_content_ext rows, including columns added after startup.
func (d *DB) ExtByContent(ids []int) (map[string]map[string]string, error) {
	if len(ids) == 0 {
		return map[string]map[string]string{}, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	rows, err := d.queryMaps("SELECT * FROM ay_content_ext WHERE contentid IN ("+strings.Join(ph, ",")+")", args...)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]string{}
	for _, r := range rows {
		out[r["contentid"]] = r
	}
	return out, nil
}

// UpsertExt writes ext_* columns for one content. Columns may be added after startup.
func (d *DB) UpsertExt(contentid int, cols map[string]string) error {
	if contentid == 0 || len(cols) == 0 {
		return nil
	}
	clean := map[string]any{}
	for k, v := range cols {
		if strings.HasPrefix(k, "ext_") && Ident(k) {
			clean[k] = v
		}
	}
	if len(clean) == 0 {
		return nil
	}
	n, err := d.queryInt("SELECT COUNT(*) FROM ay_content_ext WHERE contentid=?", contentid)
	if err != nil {
		return err
	}
	if n == 0 {
		next, err := d.queryInt("SELECT COALESCE(MAX(extid),0) FROM ay_content_ext")
		if err != nil {
			return err
		}
		clean["extid"] = next + 1
		clean["contentid"] = contentid
		return d.Insert("ay_content_ext", clean)
	}
	keys := make([]string, 0, len(clean))
	for k := range clean {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	set := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+1)
	for _, k := range keys {
		cq, err := d.QuoteIdent(k)
		if err != nil {
			return err
		}
		set = append(set, cq+"=?")
		args = append(args, clean[k])
	}
	args = append(args, contentid)
	return d.execRaw("UPDATE ay_content_ext SET "+strings.Join(set, ",")+" WHERE contentid=?", args...)
}

func cellString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "1"
		}
		return "0"
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprint(t)
	}
}
