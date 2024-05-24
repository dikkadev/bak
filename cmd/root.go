package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	outputPath   string
	zipOutput    bool
	handleSingle bool
	recursive    bool
)

var rootCmd = &cobra.Command{
	Use:   "bak [files or directories]",
	Short: "A simple CLI tool for backing up files",
	Args:  cobra.MinimumNArgs(1),
	Run:   runBackup,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputPath, "path", "p", "", "Specify the output path for the backup")
	rootCmd.PersistentFlags().BoolVarP(&zipOutput, "zip", "z", false, "Compress the backup to a ZIP file")
	rootCmd.PersistentFlags().BoolVarP(&handleSingle, "single", "s", false, "Handle multiple files as single files at the first level")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "Handle all files as single files recursively")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runBackup(cmd *cobra.Command, args []string) {
	if recursive {
		fmt.Println("Warning: Recursive backup may be heavy for many nested files. Press 'Enter' to continue or 'Ctrl+C' to cancel.")
		fmt.Scanln()
	}

	if len(args) == 1 {
		handlePath(args[0])
	} else {
		backupMultipleFiles(args)
	}
}

func handlePath(path string) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if info.IsDir() {
		backupDirectory(path)
	} else {
		backupSingleFile(path)
	}
}

func backupSingleFile(filePath string) {
	output := filePath + ".BAK"
	if zipOutput {
		output += ".zip"
		zipSingleFile(filePath, output)
	} else {
		copyFile(filePath, output)
	}
}

func backupDirectory(dirPath string) {
	if outputPath == "" {
		outputPath = "backup"
		if zipOutput {
			outputPath += ".zip"
		} else {
			outputPath += ".tar"
		}
	}

	if zipOutput {
		zipDirectory(dirPath, outputPath)
	} else {
		tarDirectory(dirPath, outputPath)
	}
}

func backupMultipleFiles(paths []string) {
	if outputPath == "" {
		outputPath = "backup"
		if zipOutput {
			outputPath += ".zip"
		} else {
			outputPath += ".tar"
		}
	}

	if zipOutput {
		zipMultipleFiles(paths, outputPath)
	} else {
		tarMultipleFiles(paths, outputPath)
	}
}

func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("File %s backed up to %s\n", src, dst)
}

func zipSingleFile(src, dst string) {
	outFile, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	inFile, err := os.Open(src)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer inFile.Close()

	w, err := zipWriter.Create(filepath.Base(src))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	_, err = io.Copy(w, inFile)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("File %s backed up to %s\n", src, dst)
}

func tarDirectory(dirPath, dst string) {
	outFile, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	err = filepath.Walk(dirPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(strings.Replace(file, dirPath, "", -1), string(filepath.Separator))

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Directory %s backed up to %s\n", dirPath, dst)
}

func zipDirectory(dirPath, dst string) {
	outFile, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	err = filepath.Walk(dirPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(strings.Replace(file, dirPath, "", -1), string(filepath.Separator))
		if fi.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(writer, f)
		return err
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Directory %s backed up to %s\n", dirPath, dst)
}

func tarMultipleFiles(paths []string, dst string) {
	outFile, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for _, path := range paths {
		err := addFileToTar(tarWriter, path, "")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	fmt.Printf("Files backed up to %s\n", dst)
}

func zipMultipleFiles(paths []string, dst string) {
	outFile, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	for _, path := range paths {
		err := addFileToZip(zipWriter, path, "")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	fmt.Printf("Files backed up to %s\n", dst)
}

func addFileToTar(tw *tar.Writer, path, baseDir string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	var base string
	if baseDir == "" {
		base = filepath.Base(path)
	} else {
		base = filepath.Join(baseDir, filepath.Base(path))
	}

	if info.IsDir() {
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, file := range files {
			err := addFileToTar(tw, filepath.Join(path, file.Name()), base)
			if err != nil {
				return err
			}
		}
	} else {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = base

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, path, baseDir string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	var base string
	if baseDir == "" {
		base = filepath.Base(path)
	} else {
		base = filepath.Join(baseDir, filepath.Base(path))
	}

	if info.IsDir() {
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, file := range files {
			err := addFileToZip(zw, filepath.Join(path, file.Name()), base)
			if err != nil {
				return err
			}
		}
	} else {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		w, err := zw.Create(base)
		if err != nil {
			return err
		}

		_, err = io.Copy(w, file)
		if err != nil {
			return err
		}
	}

	return nil
}
