package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type PackageDB interface {
	Install(pkg *InstalledPackage) error
	Get(name string) (*InstalledPackage, error)
	List() ([]*InstalledPackage, error)
	Remove(name string) error
	IsInstalled(name string) bool

	GetDependents(packageName string) ([]*InstalledPackage, error)
	GetDependencies(packageName string) ([]*InstalledPackage, error)

	Close() error
}

type SQLitePackageDB struct {
	db   *sql.DB
	path string
}

func NewSQLitePackageDB(dbPath string) (*SQLitePackageDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	pkgDB := &SQLitePackageDB{
		db:   db,
		path: dbPath,
	}

	if err := pkgDB.initSchema(); err != nil {
		return nil, err
	}

	return pkgDB, err
}

func (db *SQLitePackageDB) initSchema() error {
	schema := `
    CREATE TABLE IF NOT EXISTS packages (
        name TEXT PRIMARY KEY,
        version TEXT NOT NULL,
        description TEXT,
        homepage TEXT,
        license TEXT,
        install_path TEXT NOT NULL,
        install_date DATETIME NOT NULL
    );

    CREATE TABLE IF NOT EXISTS package_files (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        package_name TEXT NOT NULL,
        file_path TEXT NOT NULL,
        FOREIGN KEY (package_name) REFERENCES packages(name) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS dependencies (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        package_name TEXT NOT NULL,
        dependency_name TEXT NOT NULL,
        version_constraint TEXT, -- ">=2.0", "^1.5.0"
        FOREIGN KEY (package_name) REFERENCES packages(name) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS package_scripts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        package_name TEXT NOT NULL,
        script_type TEXT NOT NULL, -- "pre_install", "post_install", etc.
        script_content TEXT NOT NULL,
        FOREIGN KEY (package_name) REFERENCES packages(name) ON DELETE CASCADE
    );
    `

	_, err := db.db.Exec(schema)
	return err
}

func (db *SQLitePackageDB) Get(name string) (*InstalledPackage, error) {
	pkg := &InstalledPackage{}

	err := db.db.QueryRow(`
        SELECT name, version, description, homepage, license, install_path, install_date
        FROM packages WHERE name = ?`, name).Scan(
		&pkg.Name,
		&pkg.Version,
		&pkg.Description,
		&pkg.Homepage,
		&pkg.License,
		&pkg.InstallPath,
		&pkg.InstallDate,
	)
	if err != nil {
		return nil, err
	}

	depRows, err := db.db.Query("SELECT file_path FROM package_files WHERE package_name = ?", name)
	if err != nil {
		return nil, err
	}
	defer depRows.Close()

	for depRows.Next() {
		var filePath string
		if err := depRows.Scan(&filePath); err != nil {
			return nil, err
		}
		pkg.InstalledFiles = append(pkg.InstalledFiles, filePath)
	}

	pkg.Dependencies = make(map[string]string)
	depRows, err = db.db.Query("SELECT dependency_name, version_constraint FROM dependencies WHERE package_name = ?", name)
	if err != nil {
		return nil, err
	}
	defer depRows.Close()

	for depRows.Next() {
		var depName, constraint string
		if err := depRows.Scan(&depName, &constraint); err != nil {
			return nil, err
		}
		pkg.Dependencies[depName] = constraint
	}

	return pkg, nil
}

func (db *SQLitePackageDB) List() ([]*InstalledPackage, error) {
	rows, err := db.db.Query("SELECT name FROM packages ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*InstalledPackage
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		pkg, err := db.Get(name)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (db *SQLitePackageDB) Remove(name string) error {
	// SQLite handles cascade deletes for related tables in the above schema
	_, err := db.db.Exec("DELETE FROM packages WHERE name = ?", name)
	return err
}

func (db *SQLitePackageDB) IsInstalled(name string) bool {
	var exists bool
	err := db.db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE name = ?)", name).Scan(&exists)
	return err == nil && exists
}

func (db *SQLitePackageDB) GetDependents(packageName string) ([]*InstalledPackage, error) {
	rows, err := db.db.Query(`
        SELECT DISTINCT p.name FROM packages p
        JOIN dependencies d on p.name = d.package_name
        WHERE d.dependency_name = ?`, packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependents []*InstalledPackage
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}

		pkg, err := db.Get(name)
		if err != nil {
			return nil, err
		}
		dependents = append(dependents, pkg)
	}

	return dependents, nil
}

func (db *SQLitePackageDB) GetDependencies(packageName string) ([]*InstalledPackage, error) {
	rows, err := db.db.Query("SELECT dependency_name FROM dependencies WHERE package_name = ?", packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependencies []*InstalledPackage
	for rows.Next() {
		var depName string
		if err := rows.Scan(&depName); err != nil {
			return nil, err
		}

		if db.IsInstalled(depName) {
			pkg, err := db.Get(depName)
			if err != nil {
				return nil, err
			}
			dependencies = append(dependencies, pkg)
		}
	}

	return dependencies, nil
}

func (db *SQLitePackageDB) Install(pkg *InstalledPackage) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
        INSERT OR REPLACE INTO packages
        (name, version, description, homepage, license, install_path, install_date)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		pkg.Name,
		pkg.Version,
		pkg.Description,
		pkg.Homepage,
		pkg.License,
		pkg.InstallPath,
		pkg.InstallDate,
	)
	if err != nil {
		return err
	}

	for _, filePath := range pkg.InstalledFiles {
		_, err := tx.Exec(`
            INSERT INTO package_files (package_name, file_path)
            VALUES (?, ?)`, pkg.Name, filePath)
		if err != nil {
			return err
		}
	}

	for depName, constraint := range pkg.Dependencies {
		_, err := tx.Exec(`
            INSERT INTO dependencies (package_name, dependency_name, version_constraint)
            VALUES (?, ?, ?)`, pkg.Name, depName, constraint)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *SQLitePackageDB) Close() error {
	return db.db.Close()
}
