package sensors

import (
	"fmt"
	"os"
	"path/filepath"
)

const checkstyleTemplate = `<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
          "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
          "https://checkstyle.org/dtds/configuration_1_3.dtd">
<module name="Checker">
  <property name="severity" value="warning"/>
  <module name="TreeWalker">
    <!-- Cyclomatic Complexity Limit: max %d -->
    <module name="CyclomaticComplexity">
      <property name="max" value="%d"/>
    </module>
    <!-- Method Parameter (Argument) Count Limit: max %d -->
    <module name="ParameterNumber">
      <property name="max" value="%d"/>
    </module>
    <!-- Method Length (Function Length) Limit: max %d lines -->
    <module name="MethodLength">
      <property name="max" value="%d"/>
      <property name="countEmpty" value="false"/>
    </module>
    <!-- File Length Limit: max %d lines -->
    <module name="FileLength">
      <property name="max" value="%d"/>
    </module>
  </module>
</module>
`

func bootstrapJava(absPath string) error {
	checkPath := filepath.Join(absPath, "checkstyle.xml")
	if _, err := os.Stat(checkPath); err == nil {
		printExistingConfigBanner("checkstyle.xml", fmt.Sprintf(`
- <module name="CyclomaticComplexity"> <property name="max" value="%d"/> </module>
- <module name="ParameterNumber"> <property name="max" value="%d"/> </module>
- <module name="MethodLength"> <property name="max" value="%d"/> </module>`, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength))
	} else {
		if err := os.WriteFile(checkPath, []byte(fmt.Sprintf(checkstyleTemplate, BaselineComplexity, BaselineComplexity, BaselineArgumentCount, BaselineArgumentCount, BaselineFunctionLength, BaselineFunctionLength, BaselineFileLength, BaselineFileLength)), 0644); err != nil {
			return fmt.Errorf("failed to write checkstyle.xml: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] checkstyle.xml (Pristine Java Checkstyle Complexity Rules)\n\n")
	}
	printInstallerInstructions("java")
	return nil
}

func printJavaInstaller() {
	fmt.Fprintf(os.Stderr, "To run Java Checkstyle checks, add the checkstyle-plugin to your Maven pom.xml or Gradle build script:\n\n")
	fmt.Fprintf(os.Stderr, "Maven pom.xml Configuration:\n")
	fmt.Fprintf(os.Stderr, "  <plugin>\n")
	fmt.Fprintf(os.Stderr, "    <groupId>org.apache.maven.plugins</groupId>\n")
	fmt.Fprintf(os.Stderr, "    <artifactId>maven-checkstyle-plugin</artifactId>\n")
	fmt.Fprintf(os.Stderr, "    <version>3.3.1</version>\n")
	fmt.Fprintf(os.Stderr, "    <configuration>\n")
	fmt.Fprintf(os.Stderr, "      <configLocation>checkstyle.xml</configLocation>\n")
	fmt.Fprintf(os.Stderr, "    </configuration>\n")
	fmt.Fprintf(os.Stderr, "  </plugin>\n")
}
