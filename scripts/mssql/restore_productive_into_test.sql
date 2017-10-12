USE [master]
GO

-- create backup of prodictive database
BACKUP DATABASE [dummy] TO DISK = '/var/opt/mssql/data/dummy.bak' 
WITH CHECKSUM, COPY_ONLY, FORMAT, INIT, STATS = 10;
GO

-- drop test database if exists
IF DB_ID('dummy_test') IS NOT NULL
BEGIN
  ALTER DATABASE [dummy_test] SET SINGLE_USER WITH
  ROLLBACK IMMEDIATE;

  DROP DATABASE [dummy_test];
END
Go

-- restore productive backup into test database
RESTORE DATABASE [dummy_test]
	FROM DISK = N'/var/opt/mssql/data/dummy.bak'
	WITH MOVE 'dummy' TO '/var/opt/mssql/data/dummy_test.mdf',
	MOVE 'dummy_log' TO '/var/opt/mssql/data/dummy_test.ldf',
	RECOVERY;
GO

-- modify logical filename
ALTER DATABASE [dummy_test] MODIFY FILE ( NAME = dummy, NEWNAME = dummy_test );
GO
ALTER DATABASE [dummy_test] MODIFY FILE ( NAME = dummy_log, NEWNAME = dummy_test_log );
GO
