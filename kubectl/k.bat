@echo off

set params=

:param
set tmp=%1
if "%tmp%"=="" (
    goto end
)
set params=%params% %tmp%
shift /0
goto param

:end


kubectl --kubeconfig  ./hw  %params%