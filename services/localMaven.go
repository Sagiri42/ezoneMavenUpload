package services

import (
	"ezoneMavenUpload/utils/logger"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type MavenInfo struct {
	GroupId    string
	ArtifactId string
	Version    string
}

type MavenInfoPath struct {
	Path string
	MavenInfo
}

func FindMavenRepos(repoPath string) (mip []MavenInfoPath, err error) {
	// 将地址转为绝对路径
	if repoPath, err = filepath.Abs(repoPath); err != nil {
		logger.Logger.Error(fmt.Sprintf("无法转化地址: %s; 错误: %s", repoPath, err))
		return
	}
	var wg sync.WaitGroup
	ch := make(chan struct{}, runtime.NumCPU())
	err = fs.WalkDir(os.DirFS(repoPath), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".pom" {
			return nil
		}
		ch <- struct{}{}
		wg.Add(1)
		go func(path string) {
			if pkg := parsePkg(repoPath, path); pkg != nil {
				mip = append(mip, *pkg)
			}
			<-ch
			wg.Done()
		}(path)
		return err
	})
	wg.Wait()
	close(ch)
	return
}

func parsePkg(basePath, pkgPath string) *MavenInfoPath {
	pkgPath = filepath.Join(pkgPath)
	for _, ent := range []string{"jar", "aar", "pom"} {
		path := fmt.Sprint(basePath, string(filepath.Separator), pkgPath[:len(pkgPath)-3], ent)
		if _, err := os.Stat(path); err == nil {
			pkgPL := strings.Split(pkgPath, string(filepath.Separator))
			if len(pkgPL) < 3 {
				logger.Logger.Warn(fmt.Sprintf("不是标准的仓库文件: %s", path))
				return nil
			}
			return &MavenInfoPath{
				Path: path,
				MavenInfo: MavenInfo{
					GroupId:    strings.Join(pkgPL[:len(pkgPL)-3], "."),
					ArtifactId: pkgPL[len(pkgPL)-3],
					Version:    pkgPL[len(pkgPL)-2],
				},
			}
		}
	}
	return nil
}
