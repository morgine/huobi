package model

type Section struct {
	ID          int
	Buy10       int64
	Sell10      int64
	Inflow10    int64
	Buy30       int64
	Sell30      int64
	Inflow30    int64
	Buy60       int64
	Sell60      int64
	Inflow60    int64
	Buy300      int64
	Sell300     int64
	Inflow300   int64
	Buy900      int64
	Sell900     int64
	Inflow900   int64
	Buy3600     int64
	Sell3600    int64
	Inflow3600  int64
	Buy14400    int64
	Sell14400   int64
	Inflow14400 int64
	EndTime     int64
}

func (db *DB) CreateSection(s *Section) {
	db.db.Create(s)
}

func (db *DB) CountSection() (total int64) {
	db.db.Model(&Section{}).Count(&total)
	return
}

func (db *DB) FindSections(limit, offset int) (sections []*Section) {
	db.db.Limit(limit).Offset(offset).Find(&sections)
	return
}
